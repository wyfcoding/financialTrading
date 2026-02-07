package domain

import (
	"context"
)

// SimulationRepository 模拟仓储接口
type SimulationRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	Save(ctx context.Context, s *Simulation) error
	Get(ctx context.Context, scenarioID string) (*Simulation, error)
	List(ctx context.Context, limit int) ([]*Simulation, error)
	ListRunning(ctx context.Context, limit int) ([]*Simulation, error)
}
