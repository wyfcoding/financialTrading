package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"gorm.io/gorm"
)

type riskLimitRepository struct {
	db *gorm.DB
}

func NewRiskLimitRepository(db *gorm.DB) domain.RiskLimitRepository {
	return &riskLimitRepository{db: db}
}

func (r *riskLimitRepository) Save(ctx context.Context, limit *domain.RiskLimit) error {
	var existing domain.RiskLimit
	err := r.db.WithContext(ctx).Where("user_id = ? AND limit_type = ?", limit.UserID, limit.LimitType).First(&existing).Error
	if err == nil {
		existing.LimitValue = limit.LimitValue
		existing.CurrentValue = limit.CurrentValue
		existing.IsExceeded = limit.IsExceeded
		return r.db.WithContext(ctx).Save(&existing).Error
	}
	return r.db.WithContext(ctx).Create(limit).Error
}

func (r *riskLimitRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.RiskLimit, error) {
	var limits []*domain.RiskLimit
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&limits).Error
	return limits, err
}

func (r *riskLimitRepository) GetByUserIDAndType(ctx context.Context, userID, limitType string) (*domain.RiskLimit, error) {
	var limit domain.RiskLimit
	err := r.db.WithContext(ctx).Where("user_id = ? AND limit_type = ?", userID, limitType).First(&limit).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &limit, err
}

func (r *riskLimitRepository) Get(ctx context.Context, id string) (*domain.RiskLimit, error) {
	var limit domain.RiskLimit
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&limit).Error
	return &limit, err
}

// --- Assessment Repository ---

type riskAssessmentRepository struct {
	db *gorm.DB
}

func NewRiskAssessmentRepository(db *gorm.DB) domain.RiskAssessmentRepository {
	return &riskAssessmentRepository{db: db}
}

func (r *riskAssessmentRepository) Save(ctx context.Context, assessment *domain.RiskAssessment) error {
	return r.db.WithContext(ctx).Save(assessment).Error
}

func (r *riskAssessmentRepository) Get(ctx context.Context, id string) (*domain.RiskAssessment, error) {
	var assessment domain.RiskAssessment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&assessment).Error
	return &assessment, err
}

func (r *riskAssessmentRepository) GetLatestByUser(ctx context.Context, userID string) (*domain.RiskAssessment, error) {
	var assessment domain.RiskAssessment
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").First(&assessment).Error
	return &assessment, err
}

// --- Metrics Repository ---

type riskMetricsRepository struct {
	db *gorm.DB
}

func NewRiskMetricsRepository(db *gorm.DB) domain.RiskMetricsRepository {
	return &riskMetricsRepository{db: db}
}

func (r *riskMetricsRepository) Save(ctx context.Context, metrics *domain.RiskMetrics) error {
	return r.db.WithContext(ctx).Save(metrics).Error
}

func (r *riskMetricsRepository) Get(ctx context.Context, userID string) (*domain.RiskMetrics, error) {
	var metrics domain.RiskMetrics
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&metrics).Error
	return &metrics, err
}

// --- Alert Repository ---

type riskAlertRepository struct {
	db *gorm.DB
}

func NewRiskAlertRepository(db *gorm.DB) domain.RiskAlertRepository {
	return &riskAlertRepository{db: db}
}

func (r *riskAlertRepository) Save(ctx context.Context, alert *domain.RiskAlert) error {
	return r.db.WithContext(ctx).Save(alert).Error
}

func (r *riskAlertRepository) GetByUser(ctx context.Context, userID string, limit int) ([]*domain.RiskAlert, error) {
	var alerts []*domain.RiskAlert
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Limit(limit).Find(&alerts).Error
	return alerts, err
}

func (r *riskAlertRepository) DeleteByID(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.RiskAlert{}).Error
}

// --- Circuit Breaker Repository ---

type circuitBreakerRepository struct {
	db *gorm.DB
}

func NewCircuitBreakerRepository(db *gorm.DB) domain.CircuitBreakerRepository {
	return &circuitBreakerRepository{db: db}
}

func (r *circuitBreakerRepository) Save(ctx context.Context, cb *domain.CircuitBreaker) error {
	return r.db.WithContext(ctx).Save(cb).Error
}

func (r *circuitBreakerRepository) GetByUserID(ctx context.Context, userID string) (*domain.CircuitBreaker, error) {
	var cb domain.CircuitBreaker
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&cb).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &cb, err
}
