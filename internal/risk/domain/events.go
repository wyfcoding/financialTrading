package domain

import "time"

const (
	RiskAssessmentCreatedEventType        = "risk.assessment.created"
	RiskLimitUpdatedEventType             = "risk.limit.updated"
	RiskLimitExceededEventType            = "risk.limit.exceeded"
	CircuitBreakerFiredEventType          = "risk.circuit_breaker.fired"
	CircuitBreakerResetEventType          = "risk.circuit_breaker.reset"
	RiskAlertGeneratedEventType           = "risk.alert.generated"
	MarginCallEventType                   = "risk.margin.call"
	RiskMetricsUpdatedEventType           = "risk.metrics.updated"
	RiskLevelChangedEventType             = "risk.level.changed"
	PositionLiquidationTriggeredEventType = "risk.position.liquidation.triggered"
)

// RiskAssessmentCreatedEvent 风险评估创建事件
type RiskAssessmentCreatedEvent struct {
	AssessmentID      string    `json:"assessment_id"`
	UserID            string    `json:"user_id"`
	Symbol            string    `json:"symbol"`
	Side              string    `json:"side"`
	Quantity          float64   `json:"quantity"`
	Price             float64   `json:"price"`
	RiskLevel         RiskLevel `json:"risk_level"`
	RiskScore         float64   `json:"risk_score"`
	MarginRequirement float64   `json:"margin_requirement"`
	IsAllowed         bool      `json:"is_allowed"`
	Reason            string    `json:"reason"`
	CreatedAt         int64     `json:"created_at"`
	OccurredOn        time.Time `json:"occurred_on"`
}

// RiskLimitUpdatedEvent 风险限额更新事件
type RiskLimitUpdatedEvent struct {
	LimitID      string    `json:"limit_id"`
	UserID       string    `json:"user_id"`
	LimitType    string    `json:"limit_type"`
	LimitValue   float64   `json:"limit_value"`
	CurrentValue float64   `json:"current_value"`
	IsExceeded   bool      `json:"is_exceeded"`
	UpdatedAt    int64     `json:"updated_at"`
	OccurredOn   time.Time `json:"occurred_on"`
}

// RiskLimitExceededEvent 风险限额超出事件
type RiskLimitExceededEvent struct {
	LimitID      string    `json:"limit_id"`
	UserID       string    `json:"user_id"`
	LimitType    string    `json:"limit_type"`
	LimitValue   float64   `json:"limit_value"`
	CurrentValue float64   `json:"current_value"`
	ExceededBy   float64   `json:"exceeded_by"`
	OccurredAt   int64     `json:"occurred_at"`
	OccurredOn   time.Time `json:"occurred_on"`
}

// CircuitBreakerFiredEvent 熔断触发事件
type CircuitBreakerFiredEvent struct {
	UserID        string    `json:"user_id"`
	TriggerReason string    `json:"trigger_reason"`
	FiredAt       int64     `json:"fired_at"`
	AutoResetAt   int64     `json:"auto_reset_at"`
	OccurredOn    time.Time `json:"occurred_on"`
}

// CircuitBreakerResetEvent 熔断重置事件
type CircuitBreakerResetEvent struct {
	UserID      string    `json:"user_id"`
	ResetReason string    `json:"reset_reason"`
	ResetAt     int64     `json:"reset_at"`
	OccurredOn  time.Time `json:"occurred_on"`
}

// RiskAlertGeneratedEvent 风险告警生成事件
type RiskAlertGeneratedEvent struct {
	AlertID     string    `json:"alert_id"`
	UserID      string    `json:"user_id"`
	AlertType   string    `json:"alert_type"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	GeneratedAt int64     `json:"generated_at"`
	OccurredOn  time.Time `json:"occurred_on"`
}

// MarginCallEvent 追加保证金通知事件
type MarginCallEvent struct {
	UserID         string    `json:"user_id"`
	Symbol         string    `json:"symbol"`
	CurrentMargin  float64   `json:"current_margin"`
	RequiredMargin float64   `json:"required_margin"`
	Shortfall      float64   `json:"shortfall"`
	CallAt         int64     `json:"call_at"`
	OccurredOn     time.Time `json:"occurred_on"`
}

// RiskMetricsUpdatedEvent 风险指标更新事件
type RiskMetricsUpdatedEvent struct {
	UserID         string    `json:"user_id"`
	OldVaR95       float64   `json:"old_var_95"`
	NewVaR95       float64   `json:"new_var_95"`
	OldMaxDrawdown float64   `json:"old_max_drawdown"`
	NewMaxDrawdown float64   `json:"new_max_drawdown"`
	OldSharpeRatio float64   `json:"old_sharpe_ratio"`
	NewSharpeRatio float64   `json:"new_sharpe_ratio"`
	UpdatedAt      int64     `json:"updated_at"`
	OccurredOn     time.Time `json:"occurred_on"`
}

// RiskLevelChangedEvent 风险等级变更事件
type RiskLevelChangedEvent struct {
	UserID       string    `json:"user_id"`
	Symbol       string    `json:"symbol"`
	OldRiskLevel RiskLevel `json:"old_risk_level"`
	NewRiskLevel RiskLevel `json:"new_risk_level"`
	ChangeReason string    `json:"change_reason"`
	ChangedAt    int64     `json:"changed_at"`
	OccurredOn   time.Time `json:"occurred_on"`
}

// PositionLiquidationTriggeredEvent 仓位强平触发事件
type PositionLiquidationTriggeredEvent struct {
	UserID        string    `json:"user_id"`
	AccountID     string    `json:"account_id"`
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	Quantity      float64   `json:"quantity"`
	MarginLevel   float64   `json:"margin_level"`
	TriggerPrice  float64   `json:"trigger_price"`
	TriggerReason string    `json:"trigger_reason"`
	TriggeredAt   int64     `json:"triggered_at"`
	OccurredOn    time.Time `json:"occurred_on"`
}
