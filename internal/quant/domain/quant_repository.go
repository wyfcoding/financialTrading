package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// StrategyRepository 策略写模型仓储接口
type StrategyRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	Save(ctx context.Context, strategy *Strategy) error
	GetByID(ctx context.Context, id string) (*Strategy, error)
	Delete(ctx context.Context, id string) error
}

// StrategyReadRepository 策略读模型缓存
type StrategyReadRepository interface {
	Save(ctx context.Context, strategy *Strategy) error
	Get(ctx context.Context, id string) (*Strategy, error)
}

// BacktestResultRepository 回测结果写模型仓储接口
type BacktestResultRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	Save(ctx context.Context, result *BacktestResult) error
	GetByID(ctx context.Context, id string) (*BacktestResult, error)
}

// BacktestResultReadRepository 回测结果读模型缓存
type BacktestResultReadRepository interface {
	Save(ctx context.Context, result *BacktestResult) error
	Get(ctx context.Context, id string) (*BacktestResult, error)
}

// SignalRepository 信号写模型仓储接口
type SignalRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	Save(ctx context.Context, signal *Signal) error
	GetLatest(ctx context.Context, symbol string, indicator IndicatorType, period int) (*Signal, error)
}

// SignalReadRepository 信号读模型缓存
type SignalReadRepository interface {
	Save(ctx context.Context, signal *Signal) error
	GetLatest(ctx context.Context, symbol string, indicator IndicatorType, period int) (*Signal, error)
}

// QuantSearchRepository 提供基于 Elasticsearch 的策略/回测搜索
type QuantSearchRepository interface {
	IndexStrategy(ctx context.Context, strategy *Strategy) error
	IndexBacktestResult(ctx context.Context, result *BacktestResult) error
	SearchStrategies(ctx context.Context, keyword string, status StrategyStatus, limit, offset int) ([]*Strategy, int64, error)
	SearchBacktestResults(ctx context.Context, symbol string, status BacktestStatus, limit, offset int) ([]*BacktestResult, int64, error)
}

// MarketDataClient 市场数据客户端接口
type MarketDataClient interface {
	GetHistoricalData(ctx context.Context, symbol string) ([]decimal.Decimal, error)
}
