// Package mysql 提供了市场模拟场景仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SimulationModel 模拟场景数据库模型
type SimulationModel struct {
	gorm.Model
	ScenarioID   string  `gorm:"column:scenario_id;type:varchar(32);uniqueIndex;not null"`
	Name         string  `gorm:"column:name;type:varchar(100);not null"`
	Description  string  `gorm:"column:description;type:text"`
	Symbol       string  `gorm:"column:symbol;type:varchar(20);not null"`
	Type         string  `gorm:"column:type;type:varchar(20);not null"`
	Parameters   string  `gorm:"column:parameters;type:text"`
	Status       string  `gorm:"column:status;type:varchar(20);default:'STOPPED'"`
	StartTime    int64   `gorm:"column:start_time;type:bigint"`
	EndTime      int64   `gorm:"column:end_time;type:bigint"`
	InitialPrice float64 `gorm:"column:initial_price;type:decimal(20,8)"`
	Volatility   float64 `gorm:"column:volatility;type:decimal(10,4)"`
	Drift        float64 `gorm:"column:drift;type:decimal(10,4)"`
	IntervalMs   int64   `gorm:"column:interval_ms;type:bigint"`
}

func (SimulationModel) TableName() string { return "simulation_scenarios" }

// simulationRepositoryImpl 模拟场景仓储实现
type simulationRepositoryImpl struct {
	db *gorm.DB
}

// NewSimulationRepository 创建模拟场景仓储实例
func NewSimulationRepository(db *gorm.DB) domain.SimulationRepository {
	return &simulationRepositoryImpl{db: db}
}

func (r *simulationRepositoryImpl) Save(ctx context.Context, scenario *domain.Simulation) error {
	m := &SimulationModel{
		Model:        scenario.Model,
		ScenarioID:   scenario.ScenarioID,
		Name:         scenario.Name,
		Description:  scenario.Description,
		Symbol:       scenario.Symbol,
		Type:         string(scenario.Type),
		Parameters:   scenario.Parameters,
		Status:       string(scenario.Status),
		StartTime:    scenario.StartTime.UnixMilli(),
		EndTime:      scenario.EndTime.UnixMilli(),
		InitialPrice: scenario.InitialPrice,
		Volatility:   scenario.Volatility,
		Drift:        scenario.Drift,
		IntervalMs:   scenario.IntervalMs,
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "scenario_id"}},
		UpdateAll: true,
	}).Create(m).Error
	if err == nil {
		scenario.Model = m.Model
	}
	return err
}

func (r *simulationRepositoryImpl) Get(ctx context.Context, scenarioID string) (*domain.Simulation, error) {
	var m SimulationModel
	if err := r.db.WithContext(ctx).Where("scenario_id = ?", scenarioID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

func (r *simulationRepositoryImpl) List(ctx context.Context, limit int) ([]*domain.Simulation, error) {
	var models []SimulationModel
	if err := r.db.WithContext(ctx).Order("created_at desc").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	sims := make([]*domain.Simulation, len(models))
	for i, m := range models {
		sims[i] = r.toDomain(&m)
	}
	return sims, nil
}

func (r *simulationRepositoryImpl) toDomain(m *SimulationModel) *domain.Simulation {
	return &domain.Simulation{
		Model:       m.Model,
		ScenarioID:  m.ScenarioID,
		Name:        m.Name,
		Description: m.Description,
		Symbol:      m.Symbol,
		Type:        domain.SimulationType(m.Type),
		Parameters:  m.Parameters,
		Status:      domain.SimulationStatus(m.Status),
		StartTime:   time.UnixMilli(m.StartTime),
		EndTime:     time.UnixMilli(m.EndTime),
	}
}
