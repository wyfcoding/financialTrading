package elasticsearch

import (
	"context"
	"encoding/json"

	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
	search_pkg "github.com/wyfcoding/pkg/search"
)

type marketMakingSearchRepository struct {
	client        *search_pkg.Client
	strategyIndex string
	perfIndex     string
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

func NewMarketMakingSearchRepository(client *search_pkg.Client, strategyIndex, perfIndex string) domain.MarketMakingSearchRepository {
	if client == nil {
		return nil
	}
	if strategyIndex == "" {
		strategyIndex = "marketmaking_strategies"
	}
	if perfIndex == "" {
		perfIndex = "marketmaking_performance"
	}
	return &marketMakingSearchRepository{
		client:        client,
		strategyIndex: strategyIndex,
		perfIndex:     perfIndex,
	}
}

func (r *marketMakingSearchRepository) IndexStrategy(ctx context.Context, strategy *domain.QuoteStrategy) error {
	if strategy == nil {
		return nil
	}
	return r.client.Index(ctx, r.strategyIndex, strategy.Symbol, strategy)
}

func (r *marketMakingSearchRepository) IndexPerformance(ctx context.Context, performance *domain.MarketMakingPerformance) error {
	if performance == nil {
		return nil
	}
	return r.client.Index(ctx, r.perfIndex, performance.Symbol, performance)
}

func (r *marketMakingSearchRepository) SearchStrategies(ctx context.Context, status string, limit, offset int) ([]*domain.QuoteStrategy, int64, error) {
	query := buildStrategyQuery(status, limit, offset)
	var resp esSearchResponse
	if err := r.client.Search(ctx, r.strategyIndex, query, &resp); err != nil {
		return nil, 0, err
	}
	strategies := make([]*domain.QuoteStrategy, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var s domain.QuoteStrategy
		if err := json.Unmarshal(hit.Source, &s); err != nil {
			continue
		}
		strategies = append(strategies, &s)
	}
	return strategies, resp.Hits.Total.Value, nil
}

func (r *marketMakingSearchRepository) SearchPerformances(ctx context.Context, symbol string, limit, offset int) ([]*domain.MarketMakingPerformance, int64, error) {
	query := buildPerformanceQuery(symbol, limit, offset)
	var resp esSearchResponse
	if err := r.client.Search(ctx, r.perfIndex, query, &resp); err != nil {
		return nil, 0, err
	}
	performances := make([]*domain.MarketMakingPerformance, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var p domain.MarketMakingPerformance
		if err := json.Unmarshal(hit.Source, &p); err != nil {
			continue
		}
		performances = append(performances, &p)
	}
	return performances, resp.Hits.Total.Value, nil
}

func buildStrategyQuery(status string, limit, offset int) map[string]any {
	query := map[string]any{
		"from": offset,
		"size": limit,
		"sort": []map[string]any{{"updated_at": map[string]any{"order": "desc"}}},
	}
	if status == "" {
		query["query"] = map[string]any{"match_all": map[string]any{}}
		return query
	}
	query["query"] = map[string]any{
		"bool": map[string]any{
			"must": []map[string]any{
				{"term": map[string]any{"status": status}},
			},
		},
	}
	return query
}

func buildPerformanceQuery(symbol string, limit, offset int) map[string]any {
	query := map[string]any{
		"from": offset,
		"size": limit,
		"sort": []map[string]any{{"updated_at": map[string]any{"order": "desc"}}},
	}
	if symbol == "" {
		query["query"] = map[string]any{"match_all": map[string]any{}}
		return query
	}
	query["query"] = map[string]any{
		"bool": map[string]any{
			"must": []map[string]any{
				{"term": map[string]any{"symbol": symbol}},
			},
		},
	}
	return query
}
