// Package domain 包含市场模拟服务的领域模型
package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// SimulationType 模拟类型
type SimulationType string

const (
	SimulationTypeRandomWalk SimulationType = "RANDOM_WALK" // 随机漫步
	SimulationTypeReplay     SimulationType = "REPLAY"      // 历史回放
	SimulationTypeShock      SimulationType = "SHOCK"       // 市场冲击
)

// SimulationStatus 模拟状态
type SimulationStatus string

const (
	SimulationStatusRunning SimulationStatus = "RUNNING"
	SimulationStatusStopped SimulationStatus = "STOPPED"
)

// SimulationScenario 模拟场景实体
type SimulationScenario struct {
	gorm.Model
	ID          string           `gorm:"column:id;type:varchar(36);primaryKey" json:"id"`
	Name        string           `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Description string           `gorm:"column:description;type:text" json:"description"`
	Symbol      string           `gorm:"column:symbol;type:varchar(20);not null" json:"symbol"`
	Type        SimulationType   `gorm:"column:type;type:varchar(20);not null" json:"type"`
	Parameters  string           `gorm:"column:parameters;type:text" json:"parameters"`
	Status      SimulationStatus `gorm:"column:status;type:varchar(20);default:'STOPPED'" json:"status"`
	StartTime   time.Time        `gorm:"column:start_time;type:datetime" json:"start_time"`
	EndTime     time.Time        `gorm:"column:end_time;type:datetime" json:"end_time"`
}

// SimulationScenarioRepository 模拟场景仓储接口
type SimulationScenarioRepository interface {
	Save(ctx context.Context, scenario *SimulationScenario) error
	GetByID(ctx context.Context, id string) (*SimulationScenario, error)
}

// MarketDataPublisher 市场数据发布接口
type MarketDataPublisher interface {
	Publish(ctx context.Context, symbol string, price float64) error
}
