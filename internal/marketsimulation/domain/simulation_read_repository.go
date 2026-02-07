package domain

import "context"

// SimulationReadRepository 提供基于 Redis 的模拟场景读模型缓存
type SimulationReadRepository interface {
	Save(ctx context.Context, sim *Simulation) error
	Get(ctx context.Context, scenarioID string) (*Simulation, error)
}
