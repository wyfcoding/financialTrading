// Package mysql 提供了市场模拟场景仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SimulationScenarioModel 模拟场景数据库模型
type SimulationScenarioModel struct {
	gorm.Model
	ScenarioID  string `gorm:"column:scenario_id;type:varchar(32);uniqueIndex;not null"`
	Name        string `gorm:"column:name;type:varchar(100);not null"`
	Description string `gorm:"column:description;type:text"`
	Symbol      string `gorm:"column:symbol;type:varchar(20);not null"`
	Type        string `gorm:"column:type;type:varchar(20);not null"`
	Parameters  string `gorm:"column:parameters;type:text"`
	Status      string `gorm:"column:status;type:varchar(20);default:'STOPPED'"`
	StartTime   int64  `gorm:"column:start_time;type:bigint"`
	EndTime     int64  `gorm:"column:end_time;type:bigint"`
}

func (SimulationScenarioModel) TableName() string { return "simulation_scenarios" }

// simulationScenarioRepositoryImpl 模拟场景仓储实现
type simulationScenarioRepositoryImpl struct {
	db *gorm.DB
}

// NewSimulationScenarioRepository 创建模拟场景仓储实例
func NewSimulationScenarioRepository(db *gorm.DB) domain.SimulationScenarioRepository {
	return &simulationScenarioRepositoryImpl{db: db}
}

func (r *simulationScenarioRepositoryImpl) Save(ctx context.Context, scenario *domain.SimulationScenario) error {
	m := &SimulationScenarioModel{
		Model:       scenario.Model,
		ScenarioID:  scenario.ScenarioID,
		Name:        scenario.Name,
		Description: scenario.Description,
		Symbol:      scenario.Symbol,
		Type:        string(scenario.Type),
		Parameters:  scenario.Parameters,
		Status:      string(scenario.Status),
		StartTime:   scenario.StartTime.UnixMilli(),
		EndTime:     scenario.EndTime.UnixMilli(),
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

func (r *simulationScenarioRepositoryImpl) Get(ctx context.Context, scenarioID string) (*domain.SimulationScenario, error) {
	var m SimulationScenarioModel
	if err := r.db.WithContext(ctx).Where("scenario_id = ?", scenarioID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

func (r *simulationScenarioRepositoryImpl) toDomain(m *SimulationScenarioModel) *domain.SimulationScenario {
	return &domain.SimulationScenario{
		Model:       m.Model,
		ScenarioID:  m.ScenarioID,
		Name:        m.Name,
		Description: m.Description,
		Symbol:      m.Symbol,
		Type:        domain.SimulationType(m.Type),
		Parameters:  m.Parameters,
		Status:      domain.SimulationStatus(m.Status),
		// StartTime/EndTime mapping back to time.Time
	}
}
