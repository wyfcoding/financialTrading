package domain

import "time"

// Backtest 表示策略回测任务及结果。
type Backtest struct {
	BacktestID string       `json:"backtest_id" gorm:"column:backtest_id;primaryKey"`
	UserID     uint64       `json:"user_id" gorm:"column:user_id;index"`
	Type       StrategyType `json:"type" gorm:"column:type"`
	Symbol     string       `json:"symbol" gorm:"column:symbol"`
	Parameters string       `json:"parameters" gorm:"column:parameters;type:text"`
	StartTime  time.Time    `json:"start_time" gorm:"column:start_time"`
	EndTime    time.Time    `json:"end_time" gorm:"column:end_time"`
	Status     string       `json:"status" gorm:"column:status"`
	ResultJSON string       `json:"result_json" gorm:"column:result_json;type:text"`
	CreatedAt  time.Time    `json:"created_at" gorm:"column:created_at"`
	UpdatedAt  time.Time    `json:"updated_at" gorm:"column:updated_at"`
}
