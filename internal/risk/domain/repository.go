package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// RiskLimitRepository repository interface
type RiskLimitRepository interface {
	Save(ctx context.Context, limit *RiskLimit) error
	GetByUserID(ctx context.Context, userID string) ([]*RiskLimit, error) // Changed to return slice
	GetByUserIDAndType(ctx context.Context, userID, limitType string) (*RiskLimit, error)
	Get(ctx context.Context, id string) (*RiskLimit, error)
}

// RiskAssessmentRepository repository interface
type RiskAssessmentRepository interface {
	Save(ctx context.Context, assessment *RiskAssessment) error
	Get(ctx context.Context, id string) (*RiskAssessment, error)
	GetLatestByUser(ctx context.Context, userID string) (*RiskAssessment, error)
}

// RiskMetricsRepository repository interface
type RiskMetricsRepository interface {
	Save(ctx context.Context, metrics *RiskMetrics) error
	Get(ctx context.Context, userID string) (*RiskMetrics, error)
}

// RiskAlertRepository repository interface
type RiskAlertRepository interface {
	Save(ctx context.Context, alert *RiskAlert) error
	GetByUser(ctx context.Context, userID string, limit int) ([]*RiskAlert, error)
	DeleteByID(ctx context.Context, id string) error
}

// CircuitBreakerRepository repository interface
type CircuitBreakerRepository interface {
	Save(ctx context.Context, cb *CircuitBreaker) error
	GetByUserID(ctx context.Context, userID string) (*CircuitBreaker, error)
}

// RiskDomainService interface
type RiskDomainService interface {
	AssessTradeRisk(ctx context.Context, userID, symbol, side string, quantity, price decimal.Decimal) (*RiskAssessment, error)
	CalculateRiskMetrics(ctx context.Context, userID string) (*RiskMetrics, error)
	CheckRiskLimit(ctx context.Context, userID, limitType string) (*RiskLimit, error)
	GenerateRiskAlert(ctx context.Context, userID, alertType, severity, message string) (*RiskAlert, error)
}
