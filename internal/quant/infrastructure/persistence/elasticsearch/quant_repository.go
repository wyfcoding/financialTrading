package elasticsearch

import (
	"context"
	"encoding/json"

	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	search_pkg "github.com/wyfcoding/pkg/search"
)

type quantSearchRepository struct {
	client        *search_pkg.Client
	strategyIndex string
	backtestIndex string
}

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

func NewQuantSearchRepository(client *search_pkg.Client, strategyIndex, backtestIndex string) domain.QuantSearchRepository {
	if client == nil {
		return nil
	}
	if strategyIndex == "" {
		strategyIndex = "quant_strategies"
	}
	if backtestIndex == "" {
		backtestIndex = "quant_backtests"
	}
	return &quantSearchRepository{
		client:        client,
		strategyIndex: strategyIndex,
		backtestIndex: backtestIndex,
	}
}

func (r *quantSearchRepository) IndexStrategy(ctx context.Context, strategy *domain.Strategy) error {
	if strategy == nil {
		return nil
	}
	return r.client.Index(ctx, r.strategyIndex, strategy.ID, strategy)
}

func (r *quantSearchRepository) IndexBacktestResult(ctx context.Context, result *domain.BacktestResult) error {
	if result == nil {
		return nil
	}
	return r.client.Index(ctx, r.backtestIndex, result.ID, result)
}

func (r *quantSearchRepository) SearchStrategies(ctx context.Context, keyword string, status domain.StrategyStatus, limit, offset int) ([]*domain.Strategy, int64, error) {
	query := buildStrategyQuery(keyword, status, limit, offset)
	var resp esSearchResponse
	if err := r.client.Search(ctx, r.strategyIndex, query, &resp); err != nil {
		return nil, 0, err
	}
	strategies := make([]*domain.Strategy, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var s domain.Strategy
		if err := json.Unmarshal(hit.Source, &s); err != nil {
			continue
		}
		strategies = append(strategies, &s)
	}
	return strategies, resp.Hits.Total.Value, nil
}

func (r *quantSearchRepository) SearchBacktestResults(ctx context.Context, symbol string, status domain.BacktestStatus, limit, offset int) ([]*domain.BacktestResult, int64, error) {
	query := buildBacktestQuery(symbol, status, limit, offset)
	var resp esSearchResponse
	if err := r.client.Search(ctx, r.backtestIndex, query, &resp); err != nil {
		return nil, 0, err
	}
	results := make([]*domain.BacktestResult, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var r domain.BacktestResult
		if err := json.Unmarshal(hit.Source, &r); err != nil {
			continue
		}
		results = append(results, &r)
	}
	return results, resp.Hits.Total.Value, nil
}

func buildStrategyQuery(keyword string, status domain.StrategyStatus, limit, offset int) map[string]any {
	query := map[string]any{
		"from": offset,
		"size": limit,
		"sort": []map[string]any{{"updated_at": map[string]any{"order": "desc"}}},
	}

	must := make([]map[string]any, 0, 2)
	if keyword != "" {
		must = append(must, map[string]any{
			"multi_match": map[string]any{
				"query":  keyword,
				"fields": []string{"name", "description"},
			},
		})
	}
	if status != "" {
		must = append(must, map[string]any{
			"term": map[string]any{"status": status},
		})
	}

	if len(must) == 0 {
		query["query"] = map[string]any{"match_all": map[string]any{}}
		return query
	}
	query["query"] = map[string]any{
		"bool": map[string]any{
			"must": must,
		},
	}
	return query
}

func buildBacktestQuery(symbol string, status domain.BacktestStatus, limit, offset int) map[string]any {
	query := map[string]any{
		"from": offset,
		"size": limit,
		"sort": []map[string]any{{"updated_at": map[string]any{"order": "desc"}}},
	}

	must := make([]map[string]any, 0, 2)
	if symbol != "" {
		must = append(must, map[string]any{
			"term": map[string]any{"symbol": symbol},
		})
	}
	if status != "" {
		must = append(must, map[string]any{
			"term": map[string]any{"status": status},
		})
	}

	if len(must) == 0 {
		query["query"] = map[string]any{"match_all": map[string]any{}}
		return query
	}
	query["query"] = map[string]any{
		"bool": map[string]any{
			"must": must,
		},
	}
	return query
}
