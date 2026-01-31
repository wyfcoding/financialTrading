package domain

import "context"

// MarketMakingRepository 做市服务统一仓储接口
type MarketMakingRepository interface {
	// Strategy
	SaveStrategy(ctx context.Context, strategy *QuoteStrategy) error
	GetStrategyBySymbol(ctx context.Context, symbol string) (*QuoteStrategy, error)
	ListStrategies(ctx context.Context) ([]*QuoteStrategy, error)

	// Performance
	SavePerformance(ctx context.Context, p *MarketMakingPerformance) error
	GetPerformanceBySymbol(ctx context.Context, symbol string) (*MarketMakingPerformance, error)
}
