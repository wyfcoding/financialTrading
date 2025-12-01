// Package domain 包含量化服务的领域模型
package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// StrategyStatus 策略状态
type StrategyStatus string

const (
	StrategyStatusActive   StrategyStatus = "ACTIVE"   // 活跃
	StrategyStatusInactive StrategyStatus = "INACTIVE" // 非活跃
)

// Strategy 策略实体
// 定义量化交易策略
type Strategy struct {
	gorm.Model
	ID          string         `gorm:"column:id;type:varchar(36);primaryKey" json:"id"`
	Name        string         `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Description string         `gorm:"column:description;type:text" json:"description"`
	Script      string         `gorm:"column:script;type:text" json:"script"`
	Status      StrategyStatus `gorm:"column:status;type:varchar(20);default:'ACTIVE'" json:"status"`
	CreatedAt   time.Time      `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;type:datetime" json:"updated_at"`
}

// BacktestStatus 回测状态
type BacktestStatus string

const (
	BacktestStatusRunning   BacktestStatus = "RUNNING"
	BacktestStatusCompleted BacktestStatus = "COMPLETED"
	BacktestStatusFailed    BacktestStatus = "FAILED"
)

// BacktestResult 回测结果实体
type BacktestResult struct {
	gorm.Model
	ID          string         `gorm:"column:id;type:varchar(36);primaryKey" json:"id"`
	StrategyID  string         `gorm:"column:strategy_id;type:varchar(36);index;not null" json:"strategy_id"`
	Symbol      string         `gorm:"column:symbol;type:varchar(20);not null" json:"symbol"`
	StartTime   time.Time      `gorm:"column:start_time;type:datetime" json:"start_time"`
	EndTime     time.Time      `gorm:"column:end_time;type:datetime" json:"end_time"`
	TotalReturn float64        `gorm:"column:total_return;type:decimal(20,8)" json:"total_return"`
	MaxDrawdown float64        `gorm:"column:max_drawdown;type:decimal(20,8)" json:"max_drawdown"`
	SharpeRatio float64        `gorm:"column:sharpe_ratio;type:decimal(20,8)" json:"sharpe_ratio"`
	TotalTrades int            `gorm:"column:total_trades;type:int" json:"total_trades"`
	Status      BacktestStatus `gorm:"column:status;type:varchar(20);default:'RUNNING'" json:"status"`
	CreatedAt   time.Time      `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
}

// StrategyRepository 策略仓储接口
type StrategyRepository interface {
	Save(ctx context.Context, strategy *Strategy) error
	GetByID(ctx context.Context, id string) (*Strategy, error)
}

// BacktestResultRepository 回测结果仓储接口
type BacktestResultRepository interface {
	Save(ctx context.Context, result *BacktestResult) error
	GetByID(ctx context.Context, id string) (*BacktestResult, error)
}

// MarketDataClient 市场数据客户端接口
type MarketDataClient interface {
	GetHistoricalData(ctx context.Context, symbol string, start, end time.Time) ([]float64, error)
}
