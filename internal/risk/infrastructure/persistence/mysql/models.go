package mysql

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"gorm.io/gorm"
)

// RiskAssessmentModel MySQL 风险评估表映射
type RiskAssessmentModel struct {
	gorm.Model
	ID                string          `gorm:"primaryKey;type:varchar(36);column:id"`
	UserID            string          `gorm:"column:user_id;type:varchar(36);index;not null"`
	Symbol            string          `gorm:"column:symbol;type:varchar(20);not null"`
	Side              string          `gorm:"column:side;type:varchar(10);not null"`
	Quantity          decimal.Decimal `gorm:"column:quantity;type:decimal(20,8);not null"`
	Price             decimal.Decimal `gorm:"column:price;type:decimal(20,8);not null"`
	RiskLevel         string          `gorm:"column:risk_level;type:varchar(20);not null"`
	RiskScore         decimal.Decimal `gorm:"column:risk_score;type:decimal(5,2);not null"`
	MarginRequirement decimal.Decimal `gorm:"column:margin_requirement;type:decimal(20,8);not null"`
	IsAllowed         bool            `gorm:"column:is_allowed;type:boolean;not null"`
	Reason            string          `gorm:"column:reason;type:text"`
}

func (RiskAssessmentModel) TableName() string { return "risk_assessments" }

// RiskMetricsModel MySQL 风险指标表映射
type RiskMetricsModel struct {
	gorm.Model
	UserID      string          `gorm:"column:user_id;type:varchar(36);uniqueIndex;not null"`
	VaR95       decimal.Decimal `gorm:"column:var_95;type:decimal(20,8);not null"`
	VaR99       decimal.Decimal `gorm:"column:var_99;type:decimal(20,8);not null"`
	MaxDrawdown decimal.Decimal `gorm:"column:max_drawdown;type:decimal(20,8);not null"`
	SharpeRatio decimal.Decimal `gorm:"column:sharpe_ratio;type:decimal(20,8);not null"`
	Correlation decimal.Decimal `gorm:"column:correlation;type:decimal(20,8);not null"`
}

func (RiskMetricsModel) TableName() string { return "risk_metrics" }

// RiskLimitModel MySQL 风险限额表映射
type RiskLimitModel struct {
	gorm.Model
	ID           string          `gorm:"primaryKey;type:varchar(36);column:id"`
	UserID       string          `gorm:"column:user_id;type:varchar(36);index;not null"`
	LimitType    string          `gorm:"column:limit_type;type:varchar(50);not null"`
	LimitValue   decimal.Decimal `gorm:"column:limit_value;type:decimal(20,8);not null"`
	CurrentValue decimal.Decimal `gorm:"column:current_value;type:decimal(20,8);not null"`
	IsExceeded   bool            `gorm:"column:is_exceeded;type:boolean;not null"`
}

func (RiskLimitModel) TableName() string { return "risk_limits" }

// RiskAlertModel MySQL 风险告警表映射
type RiskAlertModel struct {
	gorm.Model
	ID        string `gorm:"primaryKey;type:varchar(36);column:id"`
	UserID    string `gorm:"column:user_id;type:varchar(36);index;not null"`
	AlertType string `gorm:"column:alert_type;type:varchar(50);not null"`
	Severity  string `gorm:"column:severity;type:varchar(20);not null"`
	Message   string `gorm:"column:message;type:text;not null"`
}

func (RiskAlertModel) TableName() string { return "risk_alerts" }

// CircuitBreakerModel MySQL 风险熔断表映射
type CircuitBreakerModel struct {
	gorm.Model
	UserID        string `gorm:"column:user_id;type:varchar(36);uniqueIndex;not null"`
	IsFired       bool   `gorm:"column:is_fired;type:boolean;not null"`
	TriggerReason string `gorm:"column:trigger_reason;type:text"`
	FiredAt       *int64 `gorm:"column:fired_at;type:bigint"`
	AutoResetAt   *int64 `gorm:"column:auto_reset_at;type:bigint"`
}

func (CircuitBreakerModel) TableName() string { return "circuit_breakers" }

// --- mapping helpers ---

func toRiskAssessmentModel(a *domain.RiskAssessment) *RiskAssessmentModel {
	if a == nil {
		return nil
	}
	return &RiskAssessmentModel{
		ID:                a.ID,
		UserID:            a.UserID,
		Symbol:            a.Symbol,
		Side:              a.Side,
		Quantity:          a.Quantity,
		Price:             a.Price,
		RiskLevel:         string(a.RiskLevel),
		RiskScore:         a.RiskScore,
		MarginRequirement: a.MarginRequirement,
		IsAllowed:         a.IsAllowed,
		Reason:            a.Reason,
	}
}

func toRiskAssessment(m *RiskAssessmentModel) *domain.RiskAssessment {
	if m == nil {
		return nil
	}
	return &domain.RiskAssessment{
		ID:                m.ID,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		UserID:            m.UserID,
		Symbol:            m.Symbol,
		Side:              m.Side,
		Quantity:          m.Quantity,
		Price:             m.Price,
		RiskLevel:         domain.RiskLevel(m.RiskLevel),
		RiskScore:         m.RiskScore,
		MarginRequirement: m.MarginRequirement,
		IsAllowed:         m.IsAllowed,
		Reason:            m.Reason,
	}
}

func toRiskMetricsModel(m *domain.RiskMetrics) *RiskMetricsModel {
	if m == nil {
		return nil
	}
	return &RiskMetricsModel{
		UserID:      m.UserID,
		VaR95:       m.VaR95,
		VaR99:       m.VaR99,
		MaxDrawdown: m.MaxDrawdown,
		SharpeRatio: m.SharpeRatio,
		Correlation: m.Correlation,
	}
}

func toRiskMetrics(m *RiskMetricsModel) *domain.RiskMetrics {
	if m == nil {
		return nil
	}
	return &domain.RiskMetrics{
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		UserID:      m.UserID,
		VaR95:       m.VaR95,
		VaR99:       m.VaR99,
		MaxDrawdown: m.MaxDrawdown,
		SharpeRatio: m.SharpeRatio,
		Correlation: m.Correlation,
	}
}

func toRiskLimitModel(l *domain.RiskLimit) *RiskLimitModel {
	if l == nil {
		return nil
	}
	return &RiskLimitModel{
		ID:           l.ID,
		UserID:       l.UserID,
		LimitType:    l.LimitType,
		LimitValue:   l.LimitValue,
		CurrentValue: l.CurrentValue,
		IsExceeded:   l.IsExceeded,
	}
}

func toRiskLimit(m *RiskLimitModel) *domain.RiskLimit {
	if m == nil {
		return nil
	}
	return &domain.RiskLimit{
		ID:           m.ID,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		UserID:       m.UserID,
		LimitType:    m.LimitType,
		LimitValue:   m.LimitValue,
		CurrentValue: m.CurrentValue,
		IsExceeded:   m.IsExceeded,
	}
}

func toRiskAlertModel(a *domain.RiskAlert) *RiskAlertModel {
	if a == nil {
		return nil
	}
	return &RiskAlertModel{
		ID:        a.ID,
		UserID:    a.UserID,
		AlertType: a.AlertType,
		Severity:  a.Severity,
		Message:   a.Message,
	}
}

func toRiskAlert(m *RiskAlertModel) *domain.RiskAlert {
	if m == nil {
		return nil
	}
	return &domain.RiskAlert{
		ID:        m.ID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		UserID:    m.UserID,
		AlertType: m.AlertType,
		Severity:  m.Severity,
		Message:   m.Message,
	}
}

func toCircuitBreakerModel(cb *domain.CircuitBreaker) *CircuitBreakerModel {
	if cb == nil {
		return nil
	}
	var firedAt *int64
	var resetAt *int64
	if cb.FiredAt != nil {
		v := cb.FiredAt.Unix()
		firedAt = &v
	}
	if cb.AutoResetAt != nil {
		v := cb.AutoResetAt.Unix()
		resetAt = &v
	}
	return &CircuitBreakerModel{
		UserID:        cb.UserID,
		IsFired:       cb.IsFired,
		TriggerReason: cb.TriggerReason,
		FiredAt:       firedAt,
		AutoResetAt:   resetAt,
	}
}

func toCircuitBreaker(m *CircuitBreakerModel) *domain.CircuitBreaker {
	if m == nil {
		return nil
	}
	var firedAt *time.Time
	var resetAt *time.Time
	if m.FiredAt != nil {
		v := time.Unix(*m.FiredAt, 0)
		firedAt = &v
	}
	if m.AutoResetAt != nil {
		v := time.Unix(*m.AutoResetAt, 0)
		resetAt = &v
	}
	return &domain.CircuitBreaker{
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		UserID:        m.UserID,
		IsFired:       m.IsFired,
		TriggerReason: m.TriggerReason,
		FiredAt:       firedAt,
		AutoResetAt:   resetAt,
	}
}
