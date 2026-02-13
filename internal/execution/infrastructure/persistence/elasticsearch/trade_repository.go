package elasticsearch

import (
	"context"
	"encoding/json"

	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	search_pkg "github.com/wyfcoding/pkg/search"
)

type tradeSearchRepository struct {
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

func NewTradeSearchRepository(client *search_pkg.Client, index string) domain.TradeSearchRepository {
	if index == "" {
		index = "trades"
	}
	return &tradeSearchRepository{
		client: client,
		index:  index,
	}
}

func (r *tradeSearchRepository) Index(ctx context.Context, trade *domain.Trade) error {
	return r.client.Index(ctx, r.index, trade.TradeID, trade)
}

func (r *tradeSearchRepository) Search(ctx context.Context, userID, symbol string, limit, offset int) ([]*domain.Trade, int64, error) {
	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{"term": map[string]any{"user_id": userID}},
				},
			},
		},
		"from": offset,
		"size": limit,
	}

	if symbol != "" {
		query["query"].(map[string]any)["bool"].(map[string]any)["must"] = append(
			query["query"].(map[string]any)["bool"].(map[string]any)["must"].([]map[string]any),
			map[string]any{"term": map[string]any{"symbol": symbol}},
		)
	}

	var resp esSearchResponse
	if err := r.client.Search(ctx, r.index, query, &resp); err != nil {
		return nil, 0, err
	}

	trades := make([]*domain.Trade, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var trade domain.Trade
		if err := json.Unmarshal(hit.Source, &trade); err != nil {
			continue
		}
		trades = append(trades, &trade)
	}

	return trades, resp.Hits.Total.Value, nil
}

func (r *tradeSearchRepository) Delete(ctx context.Context, tradeID string) error {
	return r.client.Delete(ctx, r.index, tradeID)
}
