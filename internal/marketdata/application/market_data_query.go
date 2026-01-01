package application

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

// MarketDataQuery 处理所有市场数据相关的查询操作（Queries）。
type MarketDataQuery struct {
	quoteRepo     domain.QuoteRepository
	klineRepo     domain.KlineRepository
	tradeRepo     domain.TradeRepository
	orderBookRepo domain.OrderBookRepository
}

// NewMarketDataQuery 构造函数。
func NewMarketDataQuery(
	quoteRepo domain.QuoteRepository,
	klineRepo domain.KlineRepository,
	tradeRepo domain.TradeRepository,
	orderBookRepo domain.OrderBookRepository,
) *MarketDataQuery {
	return &MarketDataQuery{
		quoteRepo:     quoteRepo,
		klineRepo:     klineRepo,
		tradeRepo:     tradeRepo,
		orderBookRepo: orderBookRepo,
	}
}

// GetLatestQuote 获取最新行情
func (q *MarketDataQuery) GetLatestQuote(ctx context.Context, symbol string) (*QuoteDTO, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	quote, err := q.quoteRepo.GetLatest(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if quote == nil {
		return nil, fmt.Errorf("quote not found")
	}

	return &QuoteDTO{
		Symbol:    quote.Symbol,
		BidPrice:  quote.BidPrice.String(),
		AskPrice:  quote.AskPrice.String(),
		BidSize:   quote.BidSize.String(),
		AskSize:   quote.AskSize.String(),
		LastPrice: quote.LastPrice.String(),
		LastSize:  quote.LastSize.String(),
		Timestamp: quote.Timestamp,
		Source:    quote.Source,
	}, nil
}

// GetHistoricalQuotes 获取历史行情
func (q *MarketDataQuery) GetHistoricalQuotes(ctx context.Context, symbol string, startTime, endTime int64) ([]*QuoteDTO, error) {
	quotes, err := q.quoteRepo.GetHistory(ctx, symbol, startTime, endTime)
	if err != nil {
		return nil, err
	}

	dtos := make([]*QuoteDTO, 0, len(quotes))
	for _, quote := range quotes {
		dtos = append(dtos, &QuoteDTO{
			Symbol:    quote.Symbol,
			BidPrice:  quote.BidPrice.String(),
			AskPrice:  quote.AskPrice.String(),
			BidSize:   quote.BidSize.String(),
			AskSize:   quote.AskSize.String(),
			LastPrice: quote.LastPrice.String(),
			LastSize:  quote.LastSize.String(),
			Timestamp: quote.Timestamp,
			Source:    quote.Source,
		})
	}
	return dtos, nil
}
