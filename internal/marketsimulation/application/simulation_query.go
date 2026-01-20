package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
)

// MarketSimulationQuery 处理所有市场模拟相关的查询操作（Queries）。
type MarketSimulationQuery struct {
	repo domain.SimulationRepository
}

// NewMarketSimulationQuery 构造函数。
func NewMarketSimulationQuery(repo domain.SimulationRepository) *MarketSimulationQuery {
	return &MarketSimulationQuery{
		repo: repo,
	}
}

// GetSimulationStatus 获取模拟状态
func (q *MarketSimulationQuery) GetSimulationStatus(ctx context.Context, scenarioID string) (*domain.Simulation, error) {
	return q.repo.Get(ctx, scenarioID)
}
