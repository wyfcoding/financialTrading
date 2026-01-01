package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// SimulationScenarioRepository 模拟场景仓储接口
type SimulationScenarioRepository interface {
	// Save 保存或更新模拟场景
	Save(ctx context.Context, scenario *SimulationScenario) error
	// Get 根据ID获取模拟场景
	Get(ctx context.Context, scenarioID string) (*SimulationScenario, error)
}

// MarketDataPublisher 市场数据发布接口
type MarketDataPublisher interface {
	// Publish 发布模拟的市场数据
	Publish(ctx context.Context, symbol string, price decimal.Decimal) error
}
