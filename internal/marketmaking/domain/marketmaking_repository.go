package domain

import "context"

// MarketMakingRepository 做市服务统一仓储接口
type MarketMakingRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	// Strategy
	SaveStrategy(ctx context.Context, strategy *QuoteStrategy) error
	GetStrategyBySymbol(ctx context.Context, symbol string) (*QuoteStrategy, error)
	ListStrategies(ctx context.Context) ([]*QuoteStrategy, error)

	// Performance
	SavePerformance(ctx context.Context, p *MarketMakingPerformance) error
	GetPerformanceBySymbol(ctx context.Context, symbol string) (*MarketMakingPerformance, error)
}

// StrategyReadRepository 做市策略读模型缓存
type StrategyReadRepository interface {
	Save(ctx context.Context, strategy *QuoteStrategy) error
	Get(ctx context.Context, symbol string) (*QuoteStrategy, error)
}

// PerformanceReadRepository 做市绩效读模型缓存
type PerformanceReadRepository interface {
	Save(ctx context.Context, performance *MarketMakingPerformance) error
	Get(ctx context.Context, symbol string) (*MarketMakingPerformance, error)
}

// MarketMakingSearchRepository 提供基于 Elasticsearch 的策略/绩效搜索
type MarketMakingSearchRepository interface {
	IndexStrategy(ctx context.Context, strategy *QuoteStrategy) error
	IndexPerformance(ctx context.Context, performance *MarketMakingPerformance) error
	SearchStrategies(ctx context.Context, status string, limit, offset int) ([]*QuoteStrategy, int64, error)
	SearchPerformances(ctx context.Context, symbol string, limit, offset int) ([]*MarketMakingPerformance, int64, error)
}
