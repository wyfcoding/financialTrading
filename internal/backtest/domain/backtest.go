package domain

import (
	"time"

	"gorm.io/gorm"
)

// BacktestTask 表示一个回测任务
type BacktestTask struct {
	gorm.Model
	TaskID         string    `gorm:"column:task_id;type:varchar(32);unique_index;not null"`
	StrategyID     string    `gorm:"column:strategy_id;type:varchar(32);not null"`
	Symbol         string    `gorm:"column:symbol;type:varchar(32);not null"`
	StartTime      time.Time `gorm:"column:start_time;not null"`
	EndTime        time.Time `gorm:"column:end_time;not null"`
	InitialCapital float64   `gorm:"column:initial_capital;type:decimal(18,4);not null"`
	Status         string    `gorm:"column:status;type:varchar(16);not null;default:'PENDING'"`
}

// BacktestReport 表示回测生成的报告
type BacktestReport struct {
	gorm.Model
	TaskID      string  `gorm:"column:task_id;type:varchar(32);unique_index;not null"`
	TotalReturn float64 `gorm:"column:total_return;type:decimal(10,4)"`
	SharpeRatio float64 `gorm:"column:sharpe_ratio;type:decimal(10,4)"`
	MaxDrawdown float64 `gorm:"column:max_drawdown;type:decimal(10,4)"`
	TotalTrades int     `gorm:"column:total_trades"`
	WinRate     float64 `gorm:"column:win_rate;type:decimal(10,4)"`
}
