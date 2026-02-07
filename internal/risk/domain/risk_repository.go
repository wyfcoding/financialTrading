package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// RiskRepository 风险聚合仓储接口（写模型）。
type RiskRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

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
	GetAlertByID(ctx context.Context, id string) (*RiskAlert, error)
	GetAlertsByUser(ctx context.Context, userID string, limit int) ([]*RiskAlert, error)
	DeleteAlertByID(ctx context.Context, id string) error

	// CircuitBreaker
	SaveCircuitBreaker(ctx context.Context, cb *CircuitBreaker) error
	GetCircuitBreakerByUserID(ctx context.Context, userID string) (*CircuitBreaker, error)
}

// RiskReadRepository 提供基于 Redis 的实时风险数据（限额、指标、熔断器）缓存
type RiskReadRepository interface {
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

// RiskSearchRepository 提供基于 Elasticsearch 的风险搜索能力
type RiskSearchRepository interface {
	IndexAssessment(ctx context.Context, assessment *RiskAssessment) error
	IndexAlert(ctx context.Context, alert *RiskAlert) error
	SearchAssessments(ctx context.Context, userID, symbol string, level RiskLevel, limit, offset int) ([]*RiskAssessment, int64, error)
	SearchAlerts(ctx context.Context, userID, severity, alertType string, limit, offset int) ([]*RiskAlert, int64, error)
}

// RiskDomainService interface
type RiskDomainService interface {
	AssessTradeRisk(ctx context.Context, userID, symbol, side string, quantity, price decimal.Decimal) (*RiskAssessment, error)
	CalculateRiskMetrics(ctx context.Context, userID string) (*RiskMetrics, error)
	CheckRiskLimit(ctx context.Context, userID, limitType string) (*RiskLimit, error)
	GenerateRiskAlert(ctx context.Context, userID, alertType, severity, message string) (*RiskAlert, error)
}
