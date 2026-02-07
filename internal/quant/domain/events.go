package domain

import "time"

const (
	StrategyCreatedEventType         = "quant.strategy.created"
	StrategyUpdatedEventType         = "quant.strategy.updated"
	StrategyDeletedEventType         = "quant.strategy.deleted"
	BacktestStartedEventType         = "quant.backtest.started"
	BacktestCompletedEventType       = "quant.backtest.completed"
	BacktestFailedEventType          = "quant.backtest.failed"
	SignalGeneratedEventType         = "quant.signal.generated"
	PortfolioOptimizedEventType      = "quant.portfolio.optimized"
	RiskAssessmentCompletedEventType = "quant.risk.assessment.completed"
)

// StrategyCreatedEvent 策略创建事件
type StrategyCreatedEvent struct {
	StrategyID  string         `json:"strategy_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Status      StrategyStatus `json:"status"`
	CreatedAt   int64          `json:"created_at"`
	OccurredOn  time.Time      `json:"occurred_on"`
}

// StrategyUpdatedEvent 策略更新事件
type StrategyUpdatedEvent struct {
	StrategyID string         `json:"strategy_id"`
	OldName    string         `json:"old_name"`
	NewName    string         `json:"new_name"`
	OldStatus  StrategyStatus `json:"old_status"`
	NewStatus  StrategyStatus `json:"new_status"`
	UpdatedAt  int64          `json:"updated_at"`
	OccurredOn time.Time      `json:"occurred_on"`
}

// StrategyDeletedEvent 策略删除事件
type StrategyDeletedEvent struct {
	StrategyID string    `json:"strategy_id"`
	Name       string    `json:"name"`
	DeletedAt  int64     `json:"deleted_at"`
	OccurredOn time.Time `json:"occurred_on"`
}

// BacktestStartedEvent 回测开始事件
type BacktestStartedEvent struct {
	BacktestID string    `json:"backtest_id"`
	StrategyID string    `json:"strategy_id"`
	Symbol     string    `json:"symbol"`
	StartTime  int64     `json:"start_time"`
	EndTime    int64     `json:"end_time"`
	StartedAt  int64     `json:"started_at"`
	OccurredOn time.Time `json:"occurred_on"`
}

// BacktestCompletedEvent 回测完成事件
type BacktestCompletedEvent struct {
	BacktestID    string    `json:"backtest_id"`
	StrategyID    string    `json:"strategy_id"`
	Symbol        string    `json:"symbol"`
	TotalReturn   float64   `json:"total_return"`
	MaxDrawdown   float64   `json:"max_drawdown"`
	SharpeRatio   float64   `json:"sharpe_ratio"`
	TotalTrades   int       `json:"total_trades"`
	WinningTrades int       `json:"winning_trades"`
	Duration      float64   `json:"duration"`
	CompletedAt   int64     `json:"completed_at"`
	OccurredOn    time.Time `json:"occurred_on"`
}

// BacktestFailedEvent 回测失败事件
type BacktestFailedEvent struct {
	BacktestID string    `json:"backtest_id"`
	StrategyID string    `json:"strategy_id"`
	Symbol     string    `json:"symbol"`
	Error      string    `json:"error"`
	ErrorCode  string    `json:"error_code"`
	FailedAt   int64     `json:"failed_at"`
	OccurredOn time.Time `json:"occurred_on"`
}

// SignalGeneratedEvent 信号生成事件
type SignalGeneratedEvent struct {
	SignalID    string        `json:"signal_id"`
	StrategyID  string        `json:"strategy_id"`
	Symbol      string        `json:"symbol"`
	Indicator   IndicatorType `json:"indicator"`
	Period      int           `json:"period"`
	Value       float64       `json:"value"`
	Confidence  float64       `json:"confidence"`
	GeneratedAt int64         `json:"generated_at"`
	OccurredOn  time.Time     `json:"occurred_on"`
}

// PortfolioOptimizedEvent 组合优化事件
type PortfolioOptimizedEvent struct {
	PortfolioID    string             `json:"portfolio_id"`
	Symbols        []string           `json:"symbols"`
	Weights        map[string]float64 `json:"weights"`
	ExpectedReturn float64            `json:"expected_return"`
	Volatility     float64            `json:"volatility"`
	SharpeRatio    float64            `json:"sharpe_ratio"`
	OptimizedAt    int64              `json:"optimized_at"`
	OccurredOn     time.Time          `json:"occurred_on"`
}

// RiskAssessmentCompletedEvent 风险评估完成事件
type RiskAssessmentCompletedEvent struct {
	AssessmentID string    `json:"assessment_id"`
	StrategyID   string    `json:"strategy_id"`
	Symbol       string    `json:"symbol"`
	VaR          float64   `json:"var"`
	CVaR         float64   `json:"cvar"`
	MaxDrawdown  float64   `json:"max_drawdown"`
	AssessmentAt int64     `json:"assessment_at"`
	OccurredOn   time.Time `json:"occurred_on"`
}
