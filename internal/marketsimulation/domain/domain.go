// Package domain 市场模拟服务的领域模型
package domain

import (
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
	// ScenarioID 场景唯一标识
	ScenarioID string `gorm:"column:scenario_id;type:varchar(32);uniqueIndex;not null" json:"scenario_id"`
	// Name 场景名称
	Name string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	// Description 场景描述
	Description string `gorm:"column:description;type:text" json:"description"`
	// Symbol 模拟的交易对
	Symbol string `gorm:"column:symbol;type:varchar(20);not null" json:"symbol"`
	// Type 模拟类型
	Type SimulationType `gorm:"column:type;type:varchar(20);not null" json:"type"`
	// Parameters 模拟参数 (JSON字符串)
	Parameters string `gorm:"column:parameters;type:text" json:"parameters"`
	// Status 模拟状态
	Status SimulationStatus `gorm:"column:status;type:varchar(20);default:'STOPPED'" json:"status"`
	// StartTime 开始时间
	StartTime time.Time `gorm:"column:start_time;type:datetime" json:"start_time"`
	// EndTime 结束时间
	EndTime time.Time `gorm:"column:end_time;type:datetime" json:"end_time"`
}

// End of domain file
