package search

import (
	"context"
	"encoding/json"

	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	search_pkg "github.com/wyfcoding/pkg/search"
)

type settlementSearchRepository struct {
	client *search_pkg.Client
	index  string
}

// esSearchResponse ES 搜索响应结构
type esSearchResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source json.RawMessage `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func NewSettlementSearchRepository(client *search_pkg.Client, index string) domain.SettlementSearchRepository {
	if index == "" {
		index = "settlements"
	}
	return &settlementSearchRepository{
		client: client,
		index:  index,
	}
}

func (r *settlementSearchRepository) Index(ctx context.Context, settlement *domain.Settlement) error {
	return r.client.Index(ctx, r.index, settlement.SettlementID, settlement)
}

func (r *settlementSearchRepository) Search(ctx context.Context, userID, symbol string, limit, offset int) ([]*domain.Settlement, int64, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]interface{}{"user_id": userID}},
				},
			},
		},
		"from": offset,
		"size": limit,
	}

	if symbol != "" {
		query["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = append(
			query["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"].([]map[string]interface{}),
			map[string]interface{}{"term": map[string]interface{}{"symbol": symbol}},
		)
	}

	var resp esSearchResponse
	if err := r.client.Search(ctx, r.index, query, &resp); err != nil {
		return nil, 0, err
	}

	settlements := make([]*domain.Settlement, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var s domain.Settlement
		if err := json.Unmarshal(hit.Source, &s); err != nil {
			continue
		}
		settlements = append(settlements, &s)
	}

	return settlements, resp.Hits.Total.Value, nil
}

func (r *settlementSearchRepository) Delete(ctx context.Context, id string) error {
	return r.client.Delete(ctx, r.index, id)
}
