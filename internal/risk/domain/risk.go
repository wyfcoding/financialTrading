// 包 风险管理服务的领域模型、实体、聚合、值对象、领域服务、仓储接口
package domain

import (
	"context"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// RiskLevel 风险等级
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "LOW"
	RiskLevelMedium   RiskLevel = "MEDIUM"
	RiskLevelHigh     RiskLevel = "HIGH"
	RiskLevelCritical RiskLevel = "CRITICAL"
)

// RiskAssessment 风险评估实体
type RiskAssessment struct {
	gorm.Model
	ID                string          `gorm:"column:id;type:varchar(36);primaryKey" json:"id"`
	UserID            string          `gorm:"column:user_id;type:varchar(36);index;not null" json:"user_id"`
	Symbol            string          `gorm:"column:symbol;type:varchar(20);not null" json:"symbol"`
	Side              string          `gorm:"column:side;type:varchar(10);not null" json:"side"`
	Quantity          decimal.Decimal `gorm:"column:quantity;type:decimal(20,8);not null" json:"quantity"`
	Price             decimal.Decimal `gorm:"column:price;type:decimal(20,8);not null" json:"price"`
	RiskLevel         RiskLevel       `gorm:"column:risk_level;type:varchar(20);not null" json:"risk_level"`
	RiskScore         decimal.Decimal `gorm:"column:risk_score;type:decimal(5,2);not null" json:"risk_score"`
	MarginRequirement decimal.Decimal `gorm:"column:margin_requirement;type:decimal(20,8);not null" json:"margin_requirement"`
	IsAllowed         bool            `gorm:"column:is_allowed;type:boolean;not null" json:"is_allowed"`
	Reason            string          `gorm:"column:reason;type:text" json:"reason"`
}

// RiskMetrics 风险指标实体
type RiskMetrics struct {
	gorm.Model
	UserID      string          `gorm:"column:user_id;type:varchar(36);uniqueIndex;not null" json:"user_id"`
	VaR95       decimal.Decimal `gorm:"column:var_95;type:decimal(20,8);not null" json:"var_95"`
	VaR99       decimal.Decimal `gorm:"column:var_99;type:decimal(20,8);not null" json:"var_99"`
	MaxDrawdown decimal.Decimal `gorm:"column:max_drawdown;type:decimal(20,8);not null" json:"max_drawdown"`
	SharpeRatio decimal.Decimal `gorm:"column:sharpe_ratio;type:decimal(20,8);not null" json:"sharpe_ratio"`
	Correlation decimal.Decimal `gorm:"column:correlation;type:decimal(20,8);not null" json:"correlation"`
}

// RiskLimit 风险限额实体
type RiskLimit struct {
	gorm.Model
	ID           string          `gorm:"column:id;type:varchar(36);primaryKey" json:"id"`
	UserID       string          `gorm:"column:user_id;type:varchar(36);index;not null" json:"user_id"`
	LimitType    string          `gorm:"column:limit_type;type:varchar(50);not null" json:"limit_type"`
	LimitValue   decimal.Decimal `gorm:"column:limit_value;type:decimal(20,8);not null" json:"limit_value"`
	CurrentValue decimal.Decimal `gorm:"column:current_value;type:decimal(20,8);not null" json:"current_value"`
	IsExceeded   bool            `gorm:"column:is_exceeded;type:boolean;not null" json:"is_exceeded"`
}

// RiskAlert 风险告警实体
type RiskAlert struct {
	gorm.Model
	ID        string `gorm:"column:id;type:varchar(36);primaryKey" json:"id"`
	UserID    string `gorm:"column:user_id;type:varchar(36);index;not null" json:"user_id"`
	AlertType string `gorm:"column:alert_type;type:varchar(50);not null" json:"alert_type"`
	Severity  string `gorm:"column:severity;type:varchar(20);not null" json:"severity"`
	Message   string `gorm:"column:message;type:text;not null" json:"message"`
}

// RiskAssessmentRepository 风险评估仓储接口
type RiskAssessmentRepository interface {
	Save(ctx context.Context, assessment *RiskAssessment) error
	Get(ctx context.Context, id string) (*RiskAssessment, error)
	GetLatestByUser(ctx context.Context, userID string) (*RiskAssessment, error)
}

// RiskMetricsRepository 风险指标仓储接口
type RiskMetricsRepository interface {
	Save(ctx context.Context, metrics *RiskMetrics) error
	Get(ctx context.Context, userID string) (*RiskMetrics, error)
}

// RiskLimitRepository 风险限额仓储接口
type RiskLimitRepository interface {
	Save(ctx context.Context, limit *RiskLimit) error
	Get(ctx context.Context, id string) (*RiskLimit, error)
	GetByUser(ctx context.Context, userID string, limitType string) (*RiskLimit, error)
}

// RiskAlertRepository 风险告警仓告接口
type RiskAlertRepository interface {
	Save(ctx context.Context, alert *RiskAlert) error
	GetByUser(ctx context.Context, userID string, limit int) ([]*RiskAlert, error)
	DeleteByID(ctx context.Context, id string) error
}

// RiskDomainService 风险领域服务
type RiskDomainService interface {
	AssessTradeRisk(ctx context.Context, userID, symbol, side string, quantity, price decimal.Decimal) (*RiskAssessment, error)
	CalculateRiskMetrics(ctx context.Context, userID string) (*RiskMetrics, error)
	CheckRiskLimit(ctx context.Context, userID, limitType string) (*RiskLimit, error)
	GenerateRiskAlert(ctx context.Context, userID, alertType, severity, message string) (*RiskAlert, error)
}
