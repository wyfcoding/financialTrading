package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type StrategyStatus string

const (
	StrategyStatusActive  StrategyStatus = "ACTIVE"
	StrategyStatusPaused  StrategyStatus = "PAUSED"
	StrategyStatusStopped StrategyStatus = "STOPPED"
)

// QuoteStrategy define a market making quote strategy
type QuoteStrategy struct {
	ID           uint            `json:"id"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Symbol       string          `json:"symbol"`
	Spread       decimal.Decimal `json:"spread"`
	MinOrderSize decimal.Decimal `json:"min_order_size"`
	MaxOrderSize decimal.Decimal `json:"max_order_size"`
	MaxPosition  decimal.Decimal `json:"max_position"`
	Status       StrategyStatus  `json:"status"`
}

type MarketMakingPerformance struct {
	ID          uint            `json:"id"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Symbol      string          `json:"symbol"`
	TotalPnL    decimal.Decimal `json:"total_pnl"`
	TotalVolume decimal.Decimal `json:"total_volume"`
	TotalTrades int64           `json:"total_trades"`
	SharpeRatio decimal.Decimal `json:"sharpe_ratio"`
	StartTime   time.Time       `json:"start_time"`
	EndTime     time.Time       `json:"end_time"`
}

func NewQuoteStrategy(symbol string, spread, minSz, maxSz, maxPos decimal.Decimal) *QuoteStrategy {
	return &QuoteStrategy{
		Symbol:       symbol,
		Spread:       spread,
		MinOrderSize: minSz,
		MaxOrderSize: maxSz,
		MaxPosition:  maxPos,
		Status:       StrategyStatusActive,
	}
}

type OrderClient interface {
	PlaceOrder(ctx context.Context, symbol string, side string, price, quantity decimal.Decimal) (string, error)
	GetPosition(ctx context.Context, symbol string) (decimal.Decimal, error)
}

type MarketDataClient interface {
	GetPrice(ctx context.Context, symbol string) (decimal.Decimal, error)
}

type AIModelClient interface {
	GetSentimentScore(ctx context.Context, text string) (float64, error)
}

func (s *QuoteStrategy) Activate() {
	s.Status = StrategyStatusActive
}

func (s *QuoteStrategy) Pause() {
	s.Status = StrategyStatusPaused
}

func (s *QuoteStrategy) UpdateConfig(spread, minSz, maxSz, maxPos decimal.Decimal) {
	s.Spread = spread
	s.MinOrderSize = minSz
	s.MaxOrderSize = maxSz
	s.MaxPosition = maxPos
}
