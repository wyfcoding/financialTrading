package domain

import (
	"time"
)

// StrategyCreatedEvent 策略创建事件
type StrategyCreatedEvent struct {
	StrategyID   string
	Name         string
	Description  string
	Status       StrategyStatus
	CreatedAt    int64
	OccurredOn   time.Time
}

// StrategyUpdatedEvent 策略更新事件
type StrategyUpdatedEvent struct {
	StrategyID   string
	OldName      string
	NewName      string
	OldStatus    StrategyStatus
	NewStatus    StrategyStatus
	UpdatedAt    int64
	OccurredOn   time.Time
}

// StrategyDeletedEvent 策略删除事件
type StrategyDeletedEvent struct {
	StrategyID   string
	Name         string
	DeletedAt    int64
	OccurredOn   time.Time
}

// BacktestStartedEvent 回测开始事件
type BacktestStartedEvent struct {
	BacktestID   string
	StrategyID   string
	Symbol       string
	StartTime    int64
	EndTime      int64
	StartedAt    int64
	OccurredOn   time.Time
}

// BacktestCompletedEvent 回测完成事件
type BacktestCompletedEvent struct {
	BacktestID   string
	StrategyID   string
	Symbol       string
	TotalReturn  float64
	MaxDrawdown  float64
	SharpeRatio  float64
	TotalTrades  int
	Duration     float64
	CompletedAt  int64
	OccurredOn   time.Time
}

// BacktestFailedEvent 回测失败事件
type BacktestFailedEvent struct {
	BacktestID   string
	StrategyID   string
	Symbol       string
	Error        string
	ErrorCode    string
	FailedAt     int64
	OccurredOn   time.Time
}

// SignalGeneratedEvent 信号生成事件
type SignalGeneratedEvent struct {
	SignalID     string
	StrategyID   string
	Symbol       string
	SignalType   string
	Price        float64
	Confidence   float64
	GeneratedAt  int64
	OccurredOn   time.Time
}

// PortfolioOptimizedEvent 组合优化事件
type PortfolioOptimizedEvent struct {
	PortfolioID  string
	Symbols      []string
	Weights      map[string]float64
	ExpectedReturn float64
	Volatility   float64
	SharpeRatio  float64
	OptimizedAt  int64
	OccurredOn   time.Time
}

// RiskAssessmentCompletedEvent 风险评估完成事件
type RiskAssessmentCompletedEvent struct {
	AssessmentID string
	StrategyID   string
	Symbol       string
	VaR          float64
	CVaR         float64
	MaxDrawdown  float64
	AssessmentAt int64
	OccurredOn   time.Time
}
