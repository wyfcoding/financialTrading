package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
	search_pkg "github.com/wyfcoding/pkg/search"
)

type marketDataSearchRepository struct {
	client     *search_pkg.Client
	quoteIndex string
	tradeIndex string
}

// esSearchResponse ES 搜索响应结构
// 与 pkg/search 客户端解析结构保持一致

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

func NewMarketDataSearchRepository(client *search_pkg.Client, quoteIndex, tradeIndex string) domain.MarketDataSearchRepository {
	if client == nil {
		return nil
	}
	if quoteIndex == "" {
		quoteIndex = "marketdata_quotes"
	}
	if tradeIndex == "" {
		tradeIndex = "marketdata_trades"
	}
	return &marketDataSearchRepository{
		client:     client,
		quoteIndex: quoteIndex,
		tradeIndex: tradeIndex,
	}
}

func (r *marketDataSearchRepository) IndexQuote(ctx context.Context, quote *domain.Quote) error {
	if quote == nil {
		return nil
	}
	id := fmt.Sprintf("%s-%d", quote.Symbol, quote.Timestamp.UnixNano())
	return r.client.Index(ctx, r.quoteIndex, id, quote)
}

func (r *marketDataSearchRepository) IndexTrade(ctx context.Context, trade *domain.Trade) error {
	if trade == nil {
		return nil
	}
	id := trade.ID
	if id == "" {
		id = fmt.Sprintf("%s-%d", trade.Symbol, trade.Timestamp.UnixNano())
	}
	return r.client.Index(ctx, r.tradeIndex, id, trade)
}

func (r *marketDataSearchRepository) SearchQuotes(ctx context.Context, symbol string, startTime, endTime time.Time, limit, offset int) ([]*domain.Quote, int64, error) {
	query := buildQuery(symbol, startTime, endTime, limit, offset)
	query["sort"] = []map[string]any{{"timestamp": map[string]any{"order": "desc"}}}

	var resp esSearchResponse
	if err := r.client.Search(ctx, r.quoteIndex, query, &resp); err != nil {
		return nil, 0, err
	}

	quotes := make([]*domain.Quote, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var quote domain.Quote
		if err := json.Unmarshal(hit.Source, &quote); err != nil {
			continue
		}
		quotes = append(quotes, &quote)
	}
	return quotes, resp.Hits.Total.Value, nil
}

func (r *marketDataSearchRepository) SearchTrades(ctx context.Context, symbol string, startTime, endTime time.Time, limit, offset int) ([]*domain.Trade, int64, error) {
	query := buildQuery(symbol, startTime, endTime, limit, offset)
	query["sort"] = []map[string]any{{"timestamp": map[string]any{"order": "desc"}}}

	var resp esSearchResponse
	if err := r.client.Search(ctx, r.tradeIndex, query, &resp); err != nil {
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

func buildQuery(symbol string, startTime, endTime time.Time, limit, offset int) map[string]any {
	must := make([]map[string]any, 0, 2)
	if symbol != "" {
		must = append(must, map[string]any{"term": map[string]any{"symbol": symbol}})
	}

	rangeCond := map[string]any{}
	if !startTime.IsZero() {
		rangeCond["gte"] = startTime.Format(time.RFC3339Nano)
	}
	if !endTime.IsZero() {
		rangeCond["lte"] = endTime.Format(time.RFC3339Nano)
	}
	if len(rangeCond) > 0 {
		must = append(must, map[string]any{"range": map[string]any{"timestamp": rangeCond}})
	}

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": must,
			},
		},
		"from": offset,
		"size": limit,
	}
	return query
}
