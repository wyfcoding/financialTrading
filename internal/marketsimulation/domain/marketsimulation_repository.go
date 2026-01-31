package domain

import (
	"context"
)

// SimulationRepository 模拟仓储接口
type SimulationRepository interface {
	Save(ctx context.Context, s *Simulation) error
	Get(ctx context.Context, scenarioID string) (*Simulation, error)
	List(ctx context.Context, limit int) ([]*Simulation, error)
}
