// 包 基础设施层实现
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/wyfcoding/financialTrading/internal/market-simulation/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
)

// SimulationScenarioModel 模拟场景数据库模型
// 对应数据库中的 simulation_scenarios 表
type SimulationScenarioModel struct {
	gorm.Model
	ID          string `gorm:"column:id;type:varchar(36);primaryKey;comment:场景ID"`
	Name        string `gorm:"column:name;type:varchar(100);not null;comment:场景名称"`
	Description string `gorm:"column:description;type:text;comment:场景描述"`
	Symbol      string `gorm:"column:symbol;type:varchar(20);not null;comment:交易对"`
	Type        string `gorm:"column:type;type:varchar(20);not null;comment:模拟类型"`
	Parameters  string `gorm:"column:parameters;type:text;comment:参数"`
	Status      string `gorm:"column:status;type:varchar(20);default:'STOPPED';comment:状态"`
}

// 指定表名
func (SimulationScenarioModel) TableName() string {
	return "simulation_scenarios"
}

// 将数据库模型转换为领域实体
func (m *SimulationScenarioModel) ToDomain() *domain.SimulationScenario {
	return &domain.SimulationScenario{
		Model:       m.Model,
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Symbol:      m.Symbol,
		Type:        domain.SimulationType(m.Type),
		Parameters:  m.Parameters,
		Status:      domain.SimulationStatus(m.Status),
		StartTime:   m.CreatedAt,
		EndTime:     m.UpdatedAt,
	}
}

// SimulationScenarioRepositoryImpl 模拟场景仓储实现
type SimulationScenarioRepositoryImpl struct {
	db *gorm.DB
}

// NewSimulationScenarioRepository 创建模拟场景仓储实例
func NewSimulationScenarioRepository(db *gorm.DB) domain.SimulationScenarioRepository {
	return &SimulationScenarioRepositoryImpl{db: db}
}

// Save 保存模拟场景
func (r *SimulationScenarioRepositoryImpl) Save(ctx context.Context, scenario *domain.SimulationScenario) error {
	model := &SimulationScenarioModel{
		Model:       scenario.Model,
		ID:          scenario.ID,
		Name:        scenario.Name,
		Description: scenario.Description,
		Symbol:      scenario.Symbol,
		Type:        string(scenario.Type),
		Parameters:  scenario.Parameters,
		Status:      string(scenario.Status),
	}

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		logging.Error(ctx, "Failed to save simulation scenario",
			"scenario_id", scenario.ID,
			"error", err,
		)
		return fmt.Errorf("failed to save simulation scenario: %w", err)
	}

	scenario.Model = model.Model
	return nil
}

// GetByID 根据 ID 获取模拟场景
func (r *SimulationScenarioRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.SimulationScenario, error) {
	var model SimulationScenarioModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get simulation scenario by ID",
			"scenario_id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get simulation scenario by ID: %w", err)
	}
	return model.ToDomain(), nil
}
