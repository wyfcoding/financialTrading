package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/quant/domain"
)

// QuantQuery 处理所有量化相关的查询操作（Queries）。
type QuantQuery struct {
	strategyRepo domain.StrategyRepository
	backtestRepo domain.BacktestResultRepository
}

// NewQuantQuery 构造函数。
func NewQuantQuery(strategyRepo domain.StrategyRepository, backtestRepo domain.BacktestResultRepository) *QuantQuery {
	return &QuantQuery{
		strategyRepo: strategyRepo,
		backtestRepo: backtestRepo,
	}
}

// GetStrategy 获取策略
func (q *QuantQuery) GetStrategy(ctx context.Context, id string) (*domain.Strategy, error) {
	return q.strategyRepo.GetByID(ctx, id)
}

// GetBacktestResult 获取回测结果
func (q *QuantQuery) GetBacktestResult(ctx context.Context, id string) (*domain.BacktestResult, error) {
	return q.backtestRepo.GetByID(ctx, id)
}
