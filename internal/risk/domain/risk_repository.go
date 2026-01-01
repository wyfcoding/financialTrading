package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

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

// CircuitBreakerRepository 风险熔断仓储接口
type CircuitBreakerRepository interface {
	Save(ctx context.Context, cb *CircuitBreaker) error
	GetByUserID(ctx context.Context, userID string) (*CircuitBreaker, error)
}

// RiskDomainService 风险领域服务
type RiskDomainService interface {
	AssessTradeRisk(ctx context.Context, userID, symbol, side string, quantity, price decimal.Decimal) (*RiskAssessment, error)
	CalculateRiskMetrics(ctx context.Context, userID string) (*RiskMetrics, error)
	CheckRiskLimit(ctx context.Context, userID, limitType string) (*RiskLimit, error)
	GenerateRiskAlert(ctx context.Context, userID, alertType, severity, message string) (*RiskAlert, error)
}
