package domain

import "time"

// BacktestTask 表示一个回测任务
type BacktestTask struct {
	TaskID         string
	StrategyID     string
	Symbol         string
	StartTime      time.Time
	EndTime        time.Time
	InitialCapital float64
	Status         string
	CreatedAt      time.Time
}

// BacktestReport 表示回测生成的报告
type BacktestReport struct {
	TaskID      string
	TotalReturn float64
	SharpeRatio float64
	MaxDrawdown float64
	TotalTrades int
	WinRate     float64
}
