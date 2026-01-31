package domain

import (
	"time"
)

// RiskAssessmentCreatedEvent 风险评估创建事件
type RiskAssessmentCreatedEvent struct {
	AssessmentID      string
	UserID            string
	Symbol            string
	Side              string
	Quantity          float64
	Price             float64
	RiskLevel         RiskLevel
	RiskScore         float64
	MarginRequirement float64
	IsAllowed         bool
	Reason            string
	CreatedAt         int64
	OccurredOn        time.Time
}

// RiskLimitExceededEvent 风险限额超出事件
type RiskLimitExceededEvent struct {
	LimitID      string
	UserID       string
	LimitType    string
	LimitValue   float64
	CurrentValue float64
	ExceededBy   float64
	OccurredAt   int64
	OccurredOn   time.Time
}

// CircuitBreakerFiredEvent 熔断触发事件
type CircuitBreakerFiredEvent struct {
	UserID        string
	TriggerReason string
	FiredAt       int64
	AutoResetAt   int64
	OccurredOn    time.Time
}

// CircuitBreakerResetEvent 熔断重置事件
type CircuitBreakerResetEvent struct {
	UserID      string
	ResetReason string
	ResetAt     int64
	OccurredOn  time.Time
}

// RiskAlertGeneratedEvent 风险告警生成事件
type RiskAlertGeneratedEvent struct {
	AlertID     string
	UserID      string
	AlertType   string
	Severity    string
	Message     string
	GeneratedAt int64
	OccurredOn  time.Time
}

// MarginCallEvent 追加保证金通知事件
type MarginCallEvent struct {
	UserID         string
	Symbol         string
	CurrentMargin  float64
	RequiredMargin float64
	Shortfall      float64
	CallAt         int64
	OccurredOn     time.Time
}

// RiskMetricsUpdatedEvent 风险指标更新事件
type RiskMetricsUpdatedEvent struct {
	UserID         string
	OldVaR95       float64
	NewVaR95       float64
	OldMaxDrawdown float64
	NewMaxDrawdown float64
	OldSharpeRatio float64
	NewSharpeRatio float64
	UpdatedAt      int64
	OccurredOn     time.Time
}

// RiskLevelChangedEvent 风险等级变更事件
type RiskLevelChangedEvent struct {
	UserID       string
	Symbol       string
	OldRiskLevel RiskLevel
	NewRiskLevel RiskLevel
	ChangeReason string
	ChangedAt    int64
	OccurredOn   time.Time
}
