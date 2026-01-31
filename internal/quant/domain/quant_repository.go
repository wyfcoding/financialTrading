package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// StrategyRepository 策略仓储接口
type StrategyRepository interface {
	Save(ctx context.Context, strategy *Strategy) error
	GetByID(ctx context.Context, id string) (*Strategy, error)
}

type SignalRepository interface {
	Save(ctx context.Context, signal *Signal) error
	GetLatest(ctx context.Context, symbol string, indicator IndicatorType, period int) (*Signal, error)
}

// BacktestResultRepository 回测结果仓储接口
type BacktestResultRepository interface {
	Save(ctx context.Context, result *BacktestResult) error
	GetByID(ctx context.Context, id string) (*BacktestResult, error)
}

// MarketDataClient 市场数据客户端接口
type MarketDataClient interface {
	GetHistoricalData(ctx context.Context, symbol string) ([]decimal.Decimal, error)
}
