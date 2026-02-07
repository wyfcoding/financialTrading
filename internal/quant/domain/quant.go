package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// StrategyStatus 策略状态
type StrategyStatus string

const (
	StrategyStatusActive   StrategyStatus = "ACTIVE"
	StrategyStatusInactive StrategyStatus = "INACTIVE"
)

// Strategy 策略实体
type Strategy struct {
	ID          string         `json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Script      string         `json:"script"`
	Status      StrategyStatus `json:"status"`
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
	ID            string          `json:"id"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	StrategyID    string          `json:"strategy_id"`
	Symbol        string          `json:"symbol"`
	StartTime     int64           `json:"start_time"`
	EndTime       int64           `json:"end_time"`
	TotalReturn   decimal.Decimal `json:"total_return"`
	MaxDrawdown   decimal.Decimal `json:"max_drawdown"`
	SharpeRatio   decimal.Decimal `json:"sharpe_ratio"`
	TotalTrades   int             `json:"total_trades"`
	WinningTrades int             `json:"winning_trades"`
	Status        BacktestStatus  `json:"status"`
}

// MilliToTime 将毫秒转换为 time.Time
func MilliToTime(milli int64) time.Time {
	return time.UnixMilli(milli)
}
