package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// RiskLevel 风险等级
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "LOW"
	RiskLevelMedium   RiskLevel = "MEDIUM"
	RiskLevelHigh     RiskLevel = "HIGH"
	RiskLevelCritical RiskLevel = "CRITICAL"
)

const (
	LimitTypeMaxPosition = "MAX_POSITION"
	LimitTypeCreditLimit = "CREDIT_LIMIT"
)

// RiskAssessment 风险评估实体（领域模型）
type RiskAssessment struct {
	ID                string          `json:"id"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	UserID            string          `json:"user_id"`
	Symbol            string          `json:"symbol"`
	Side              string          `json:"side"`
	Quantity          decimal.Decimal `json:"quantity"`
	Price             decimal.Decimal `json:"price"`
	RiskLevel         RiskLevel       `json:"risk_level"`
	RiskScore         decimal.Decimal `json:"risk_score"`
	MarginRequirement decimal.Decimal `json:"margin_requirement"`
	IsAllowed         bool            `json:"is_allowed"`
	Reason            string          `json:"reason"`
}

// RiskMetrics 风险指标实体
type RiskMetrics struct {
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	UserID      string          `json:"user_id"`
	VaR95       decimal.Decimal `json:"var_95"`
	VaR99       decimal.Decimal `json:"var_99"`
	MaxDrawdown decimal.Decimal `json:"max_drawdown"`
	SharpeRatio decimal.Decimal `json:"sharpe_ratio"`
	Correlation decimal.Decimal `json:"correlation"`
}

// RiskLimit 风险限额实体
type RiskLimit struct {
	ID           string          `json:"id"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	UserID       string          `json:"user_id"`
	LimitType    string          `json:"limit_type"`
	LimitValue   decimal.Decimal `json:"limit_value"`
	CurrentValue decimal.Decimal `json:"current_value"`
	IsExceeded   bool            `json:"is_exceeded"`
}

// RiskAlert 风险告警实体
type RiskAlert struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    string    `json:"user_id"`
	AlertType string    `json:"alert_type"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
}

// CircuitBreaker 风险熔断实体
type CircuitBreaker struct {
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	UserID        string     `json:"user_id"`
	IsFired       bool       `json:"is_fired"`
	TriggerReason string     `json:"trigger_reason"`
	FiredAt       *time.Time `json:"fired_at"`
	AutoResetAt   *time.Time `json:"auto_reset_at"`
}
