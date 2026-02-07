// Package mysql 提供了市场模拟场景仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// simulationRepository 模拟场景仓储实现
type simulationRepository struct {
	db *gorm.DB
}

// NewSimulationRepository 创建模拟场景仓储实例
func NewSimulationRepository(db *gorm.DB) domain.SimulationRepository {
	return &simulationRepository{db: db}
}

// --- tx helpers ---

func (r *simulationRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *simulationRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *simulationRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *simulationRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *simulationRepository) Save(ctx context.Context, scenario *domain.Simulation) error {
	model := toSimulationModel(scenario)
	if model == nil {
		return nil
	}
	db := r.getDB(ctx).WithContext(ctx)
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "scenario_id"}},
		UpdateAll: true,
	}).Create(model).Error; err != nil {
		return err
	}

	// Reload to ensure ID/timestamps are synced for upsert path
	if err := db.Where("scenario_id = ?", model.ScenarioID).First(model).Error; err != nil {
		return err
	}
	scenario.ID = model.ID
	scenario.CreatedAt = model.CreatedAt
	scenario.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *simulationRepository) Get(ctx context.Context, scenarioID string) (*domain.Simulation, error) {
	var m SimulationModel
	if err := r.getDB(ctx).WithContext(ctx).Where("scenario_id = ?", scenarioID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toSimulation(&m), nil
}

func (r *simulationRepository) List(ctx context.Context, limit int) ([]*domain.Simulation, error) {
	var models []SimulationModel
	if err := r.getDB(ctx).WithContext(ctx).Order("created_at desc").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	sims := make([]*domain.Simulation, len(models))
	for i := range models {
		sims[i] = toSimulation(&models[i])
	}
	return sims, nil
}

func (r *simulationRepository) ListRunning(ctx context.Context, limit int) ([]*domain.Simulation, error) {
	var models []SimulationModel
	if err := r.getDB(ctx).WithContext(ctx).
		Where("status = ?", string(domain.SimulationStatusRunning)).
		Order("created_at desc").
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, err
	}
	sims := make([]*domain.Simulation, len(models))
	for i := range models {
		sims[i] = toSimulation(&models[i])
	}
	return sims, nil
}

func (r *simulationRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
