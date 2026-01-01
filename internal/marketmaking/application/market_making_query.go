package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

// MarketMakingQuery 处理所有做市相关的查询操作（Queries）。
type MarketMakingQuery struct {
	strategyRepo    domain.QuoteStrategyRepository
	performanceRepo domain.PerformanceRepository
}

// NewMarketMakingQuery 构造函数。
func NewMarketMakingQuery(
	strategyRepo domain.QuoteStrategyRepository,
	performanceRepo domain.PerformanceRepository,
) *MarketMakingQuery {
	return &MarketMakingQuery{
		strategyRepo:    strategyRepo,
		performanceRepo: performanceRepo,
	}
}

// GetStrategy 获取做市策略
func (q *MarketMakingQuery) GetStrategy(ctx context.Context, symbol string) (*domain.QuoteStrategy, error) {
	return q.strategyRepo.GetStrategyBySymbol(ctx, symbol)
}

// GetPerformance 获取做市绩效
func (q *MarketMakingQuery) GetPerformance(ctx context.Context, symbol string) (*domain.MarketMakingPerformance, error) {
	return q.performanceRepo.GetPerformanceBySymbol(ctx, symbol)
}
