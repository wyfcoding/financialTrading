package elasticsearch

import (
	"context"
	"encoding/json"

	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
	search_pkg "github.com/wyfcoding/pkg/search"
)

type referenceDataSearchRepository struct {
	client          *search_pkg.Client
	symbolIndex     string
	exchangeIndex   string
	instrumentIndex string
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

func NewReferenceDataSearchRepository(client *search_pkg.Client, symbolIndex, exchangeIndex, instrumentIndex string) domain.ReferenceDataSearchRepository {
	if client == nil {
		return nil
	}
	if symbolIndex == "" {
		symbolIndex = "referencedata_symbols"
	}
	if exchangeIndex == "" {
		exchangeIndex = "referencedata_exchanges"
	}
	if instrumentIndex == "" {
		instrumentIndex = "referencedata_instruments"
	}
	return &referenceDataSearchRepository{
		client:          client,
		symbolIndex:     symbolIndex,
		exchangeIndex:   exchangeIndex,
		instrumentIndex: instrumentIndex,
	}
}

func (r *referenceDataSearchRepository) IndexSymbol(ctx context.Context, symbol *domain.Symbol) error {
	if symbol == nil {
		return nil
	}
	id := symbol.ID
	if id == "" {
		id = symbol.SymbolCode
	}
	return r.client.Index(ctx, r.symbolIndex, id, symbol)
}

func (r *referenceDataSearchRepository) IndexExchange(ctx context.Context, exchange *domain.Exchange) error {
	if exchange == nil {
		return nil
	}
	id := exchange.ID
	if id == "" {
		id = exchange.Name
	}
	return r.client.Index(ctx, r.exchangeIndex, id, exchange)
}

func (r *referenceDataSearchRepository) IndexInstrument(ctx context.Context, instrument *domain.Instrument) error {
	if instrument == nil {
		return nil
	}
	id := instrument.ID
	if id == "" {
		id = instrument.Symbol
	}
	return r.client.Index(ctx, r.instrumentIndex, id, instrument)
}

func (r *referenceDataSearchRepository) SearchSymbols(ctx context.Context, exchangeID, status, keyword string, limit, offset int) ([]*domain.Symbol, int64, error) {
	query := buildSymbolQuery(exchangeID, status, keyword, limit, offset)
	var resp esSearchResponse
	if err := r.client.Search(ctx, r.symbolIndex, query, &resp); err != nil {
		return nil, 0, err
	}
	results := make([]*domain.Symbol, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var s domain.Symbol
		if err := json.Unmarshal(hit.Source, &s); err != nil {
			continue
		}
		results = append(results, &s)
	}
	return results, resp.Hits.Total.Value, nil
}

func (r *referenceDataSearchRepository) SearchExchanges(ctx context.Context, name, country, status string, limit, offset int) ([]*domain.Exchange, int64, error) {
	query := buildExchangeQuery(name, country, status, limit, offset)
	var resp esSearchResponse
	if err := r.client.Search(ctx, r.exchangeIndex, query, &resp); err != nil {
		return nil, 0, err
	}
	results := make([]*domain.Exchange, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var e domain.Exchange
		if err := json.Unmarshal(hit.Source, &e); err != nil {
			continue
		}
		results = append(results, &e)
	}
	return results, resp.Hits.Total.Value, nil
}

func (r *referenceDataSearchRepository) SearchInstruments(ctx context.Context, symbol, instrumentType string, limit, offset int) ([]*domain.Instrument, int64, error) {
	query := buildInstrumentQuery(symbol, instrumentType, limit, offset)
	var resp esSearchResponse
	if err := r.client.Search(ctx, r.instrumentIndex, query, &resp); err != nil {
		return nil, 0, err
	}
	results := make([]*domain.Instrument, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var i domain.Instrument
		if err := json.Unmarshal(hit.Source, &i); err != nil {
			continue
		}
		results = append(results, &i)
	}
	return results, resp.Hits.Total.Value, nil
}

func buildSymbolQuery(exchangeID, status, keyword string, limit, offset int) map[string]any {
	must := make([]map[string]any, 0, 3)
	if exchangeID != "" {
		must = append(must, map[string]any{"term": map[string]any{"exchange_id": exchangeID}})
	}
	if status != "" {
		must = append(must, map[string]any{"term": map[string]any{"status": status}})
	}
	if keyword != "" {
		must = append(must, map[string]any{"match": map[string]any{"symbol_code": keyword}})
	}

	query := map[string]any{
		"from": offset,
		"size": limit,
		"sort": []map[string]any{{"updated_at": map[string]any{"order": "desc"}}},
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

func buildExchangeQuery(name, country, status string, limit, offset int) map[string]any {
	must := make([]map[string]any, 0, 3)
	if name != "" {
		must = append(must, map[string]any{"match": map[string]any{"name": name}})
	}
	if country != "" {
		must = append(must, map[string]any{"term": map[string]any{"country": country}})
	}
	if status != "" {
		must = append(must, map[string]any{"term": map[string]any{"status": status}})
	}

	query := map[string]any{
		"from": offset,
		"size": limit,
		"sort": []map[string]any{{"updated_at": map[string]any{"order": "desc"}}},
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

func buildInstrumentQuery(symbol, instrumentType string, limit, offset int) map[string]any {
	must := make([]map[string]any, 0, 2)
	if symbol != "" {
		must = append(must, map[string]any{"term": map[string]any{"symbol": symbol}})
	}
	if instrumentType != "" {
		must = append(must, map[string]any{"term": map[string]any{"type": instrumentType}})
	}

	query := map[string]any{
		"from": offset,
		"size": limit,
		"sort": []map[string]any{{"updated_at": map[string]any{"order": "desc"}}},
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
