package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// RiskRepository 风险聚合仓储接口，整合了限额、评估、指标、告警和熔断器的持久化操作。
type RiskRepository interface {
	// Limit
	SaveLimit(ctx context.Context, limit *RiskLimit) error
	GetLimitsByUserID(ctx context.Context, userID string) ([]*RiskLimit, error)
	GetLimitByUserIDAndType(ctx context.Context, userID, limitType string) (*RiskLimit, error)
	GetLimit(ctx context.Context, id string) (*RiskLimit, error)

	// Assessment
	SaveAssessment(ctx context.Context, assessment *RiskAssessment) error
	GetAssessment(ctx context.Context, id string) (*RiskAssessment, error)
	GetLatestAssessmentByUser(ctx context.Context, userID string) (*RiskAssessment, error)

	// Metrics
	SaveMetrics(ctx context.Context, metrics *RiskMetrics) error
	GetMetrics(ctx context.Context, userID string) (*RiskMetrics, error)

	// Alert
	SaveAlert(ctx context.Context, alert *RiskAlert) error
	GetAlertsByUser(ctx context.Context, userID string, limit int) ([]*RiskAlert, error)
	DeleteAlertByID(ctx context.Context, id string) error

	// CircuitBreaker
	SaveCircuitBreaker(ctx context.Context, cb *CircuitBreaker) error
	GetCircuitBreakerByUserID(ctx context.Context, userID string) (*CircuitBreaker, error)
}

// RiskRedisRepository 提供基于 Redis 的实时风险数据（限额、指标、熔断器）缓存
type RiskRedisRepository interface {
	SaveLimit(ctx context.Context, userID string, limit *RiskLimit) error
	GetLimit(ctx context.Context, userID, limitType string) (*RiskLimit, error)
	DeleteLimit(ctx context.Context, userID, limitType string) error

	SaveMetrics(ctx context.Context, userID string, metrics *RiskMetrics) error
	GetMetrics(ctx context.Context, userID string) (*RiskMetrics, error)
	DeleteMetrics(ctx context.Context, userID string) error

	SaveCircuitBreaker(ctx context.Context, userID string, cb *CircuitBreaker) error
	GetCircuitBreaker(ctx context.Context, userID string) (*CircuitBreaker, error)
	DeleteCircuitBreaker(ctx context.Context, userID string) error
}

// RiskDomainService interface
type RiskDomainService interface {
	AssessTradeRisk(ctx context.Context, userID, symbol, side string, quantity, price decimal.Decimal) (*RiskAssessment, error)
	CalculateRiskMetrics(ctx context.Context, userID string) (*RiskMetrics, error)
	CheckRiskLimit(ctx context.Context, userID, limitType string) (*RiskLimit, error)
	GenerateRiskAlert(ctx context.Context, userID, alertType, severity, message string) (*RiskAlert, error)
}
