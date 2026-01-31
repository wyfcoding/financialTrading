package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

// MarketMakingQuery 处理所有做市相关的查询操作（Queries）。
type MarketMakingQuery struct {
	repo domain.MarketMakingRepository
}

// NewMarketMakingQuery 构造函数。
func NewMarketMakingQuery(
	repo domain.MarketMakingRepository,
) *MarketMakingQuery {
	return &MarketMakingQuery{
		repo: repo,
	}
}

// GetStrategy 获取做市策略
func (q *MarketMakingQuery) GetStrategy(ctx context.Context, symbol string) (*domain.QuoteStrategy, error) {
	return q.repo.GetStrategyBySymbol(ctx, symbol)
}

// GetPerformance 获取做市绩效
func (q *MarketMakingQuery) GetPerformance(ctx context.Context, symbol string) (*domain.MarketMakingPerformance, error) {
	return q.repo.GetPerformanceBySymbol(ctx, symbol)
}
