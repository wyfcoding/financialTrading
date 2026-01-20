package domain

import "context"

type QuoteStrategyRepository interface {
	SaveStrategy(ctx context.Context, strategy *QuoteStrategy) error
	GetStrategyBySymbol(ctx context.Context, symbol string) (*QuoteStrategy, error)
	// GetByID if needed
}

type PerformanceRepository interface {
	SavePerformance(ctx context.Context, p *MarketMakingPerformance) error
	GetPerformanceBySymbol(ctx context.Context, symbol string) (*MarketMakingPerformance, error)
}
