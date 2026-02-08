// Package domain 算法交易服务领域层
// 生成摘要：
// 1) 定义策略聚合根
// 2) 定义策略执行引擎接口
// 3) 定义回测任务实体
package domain

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// StrategyType 策略类型
type StrategyType int8

const (
	StrategyTypeTWAP      StrategyType = 1 // 时间加权平均价格
	StrategyTypeVWAP      StrategyType = 2 // 成交量加权平均价格
	StrategyTypeGrid      StrategyType = 3 // 网格交易
	StrategyTypeArbitrage StrategyType = 4 // 套利
)

// StrategyStatus 策略状态
type StrategyStatus int8

const (
	StrategyStatusCreated StrategyStatus = 1
	StrategyStatusRunning StrategyStatus = 2
	StrategyStatusPaused  StrategyStatus = 3
	StrategyStatusStopped StrategyStatus = 4
	StrategyStatusFailed  StrategyStatus = 5
)

// Strategy 策略聚合根
type Strategy struct {
	gorm.Model
	StrategyID string         `gorm:"column:strategy_id;type:varchar(32);unique_index;not null"`
	UserID     uint64         `gorm:"column:user_id;index;not null"`
	Type       StrategyType   `gorm:"column:type;type:tinyint;not null"`
	Status     StrategyStatus `gorm:"column:status;type:tinyint;not null;default:1"`
	Symbol     string         `gorm:"column:symbol;type:varchar(32);not null"`
	Parameters string         `gorm:"column:parameters;type:json"` // JSON存储参数

	// 运行统计
	ExecutedAmount   int64 `gorm:"column:executed_amount;not null;default:0"`
	ExecutedQuantity int64 `gorm:"column:executed_quantity;not null;default:0"`

	// 领域事件
	domainEvents []DomainEvent `gorm:"-"`
}

// TableName 表名
func (Strategy) TableName() string {
	return "strategies"
}

// NewStrategy 创建策略
func NewStrategy(id string, userID uint64, sType StrategyType, symbol, params string) *Strategy {
	return &Strategy{
		StrategyID: id,
		UserID:     userID,
		Type:       sType,
		Status:     StrategyStatusCreated,
		Symbol:     symbol,
		Parameters: params,
	}
}

// Start 启动策略
func (s *Strategy) Start() error {
	if s.Status != StrategyStatusCreated && s.Status != StrategyStatusPaused && s.Status != StrategyStatusStopped {
		return errors.New("invalid status for start")
	}
	s.Status = StrategyStatusRunning

	s.addEvent(&StrategyStartedEvent{
		StrategyID: s.StrategyID,
		UserID:     s.UserID,
		Timestamp:  time.Now(),
	})

	return nil
}

// Stop 停止策略
func (s *Strategy) Stop() error {
	if s.Status != StrategyStatusRunning && s.Status != StrategyStatusPaused {
		return errors.New("invalid status for stop")
	}
	s.Status = StrategyStatusStopped

	s.addEvent(&StrategyStoppedEvent{
		StrategyID: s.StrategyID,
		UserID:     s.UserID,
		Timestamp:  time.Now(),
	})

	return nil
}

func (s *Strategy) addEvent(event DomainEvent) {
	s.domainEvents = append(s.domainEvents, event)
}

func (s *Strategy) GetDomainEvents() []DomainEvent {
	return s.domainEvents
}

func (s *Strategy) ClearDomainEvents() {
	s.domainEvents = nil
}

// Backtest 回测任务
type Backtest struct {
	gorm.Model
	BacktestID string       `gorm:"column:backtest_id;type:varchar(32);unique_index;not null"`
	UserID     uint64       `gorm:"column:user_id;index;not null"`
	Type       StrategyType `gorm:"column:type;type:tinyint;not null"`
	Symbol     string       `gorm:"column:symbol;type:varchar(32);not null"`
	Parameters string       `gorm:"column:parameters;type:json"`
	StartTime  time.Time    `gorm:"column:start_time;not null"`
	EndTime    time.Time    `gorm:"column:end_time;not null"`
	Status     string       `gorm:"column:status;type:varchar(16);not null;default:'PENDING'"`
	ResultJSON string       `gorm:"column:result_json;type:json"`
}

// TableName 表名
func (Backtest) TableName() string {
	return "backtests"
}
