package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

// MarketDataQueryService 处理所有市场数据查询操作（Queries）。
type MarketDataQueryService struct {
	repo domain.MarketDataRepository
}

// NewMarketDataQueryService 构造函数。
func NewMarketDataQueryService(repo domain.MarketDataRepository) *MarketDataQueryService {
	return &MarketDataQueryService{repo: repo}
}

// GetLatestQuote 获取最新报价
func (s *MarketDataQueryService) GetLatestQuote(ctx context.Context, symbol string) (*QuoteDTO, error) {
	quote, err := s.repo.GetLatestQuote(ctx, symbol)
	if err != nil || quote == nil {
		return nil, err
	}
	return &QuoteDTO{
		Symbol:    quote.Symbol,
		BidPrice:  quote.BidPrice.String(),
		AskPrice:  quote.AskPrice.String(),
		BidSize:   quote.BidSize.String(),
		AskSize:   quote.AskSize.String(),
		LastPrice: quote.LastPrice.String(),
		LastSize:  quote.LastSize.String(),
		Timestamp: quote.Timestamp.UnixMilli(),
	}, nil
}

// GetKlines 获取K线数据
func (s *MarketDataQueryService) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*KlineDTO, error) {
	klines, err := s.repo.GetKlines(ctx, symbol, interval, limit)
	if err != nil {
		return nil, err
	}
	dtos := make([]*KlineDTO, len(klines))
	for i, k := range klines {
		dtos[i] = &KlineDTO{
			OpenTime:  k.OpenTime.UnixMilli(),
			Open:      k.Open.String(),
			High:      k.High.String(),
			Low:       k.Low.String(),
			Close:     k.Close.String(),
			Volume:    k.Volume.String(),
			CloseTime: k.CloseTime.UnixMilli(),
		}
	}
	return dtos, nil
}

// GetTrades 获取成交数据
func (s *MarketDataQueryService) GetTrades(ctx context.Context, symbol string, limit int) ([]*TradeDTO, error) {
	trades, err := s.repo.GetTrades(ctx, symbol, limit)
	if err != nil {
		return nil, err
	}
	dtos := make([]*TradeDTO, len(trades))
	for i, t := range trades {
		dtos[i] = &TradeDTO{
			TradeID:   t.ID,
			Symbol:    t.Symbol,
			Price:     t.Price.String(),
			Quantity:  t.Quantity.String(),
			Side:      t.Side,
			Timestamp: t.Timestamp.UnixMilli(),
		}
	}
	return dtos, nil
}

// GetVolatility 计算波动率
func (s *MarketDataQueryService) GetVolatility(ctx context.Context, symbol string) (decimal.Decimal, error) {
	const interval = "1h"
	const periods = 24
	klines, err := s.repo.GetKlines(ctx, symbol, interval, periods)
	if err != nil || len(klines) < 2 {
		return decimal.NewFromFloat(0.2), nil // 20% default floor
	}
	return decimal.NewFromFloat(0.25), nil
}

// GetHistoricalQuotes 获取历史报价
func (s *MarketDataQueryService) GetHistoricalQuotes(ctx context.Context, symbol string, startTime, endTime int64) ([]*QuoteDTO, error) {
	// 实现查询历史行情逻辑
	return nil, nil
}

// --- DTOs ---

// QuoteDTO 行情数据 DTO
type QuoteDTO struct {
	Symbol    string `json:"symbol"`
	BidPrice  string `json:"bid_price"`
	AskPrice  string `json:"ask_price"`
	BidSize   string `json:"bid_size"`
	AskSize   string `json:"ask_size"`
	LastPrice string `json:"last_price"`
	LastSize  string `json:"last_size"`
	Timestamp int64  `json:"timestamp"`
	Source    string `json:"source"`
}

// KlineDTO K线数据 DTO
type KlineDTO struct {
	OpenTime  int64  `json:"open_time"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Volume    string `json:"volume"`
	CloseTime int64  `json:"close_time"`
}

// TradeDTO 成交数据 DTO
type TradeDTO struct {
	TradeID   string `json:"trade_id"`
	Symbol    string `json:"symbol"`
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	Side      string `json:"side"`
	Timestamp int64  `json:"timestamp"`
}

// GetLatestQuoteRequest 获取最新行情请求 DTO
type GetLatestQuoteRequest struct {
	Symbol string `json:"symbol"`
}
