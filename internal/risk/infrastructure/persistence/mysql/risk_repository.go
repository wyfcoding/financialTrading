package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"gorm.io/gorm"
)

type riskRepository struct {
	db *gorm.DB
}

// NewRiskRepository 创建并返回一个新的 RiskRepository 实例。
// 它实现了 domain 包中定义的所有风险相关仓储接口。
func NewRiskRepository(db *gorm.DB) *riskRepository {
	return &riskRepository{db: db}
}

// --- RiskLimitRepository 实现 ---

func (r *riskRepository) SaveLimit(ctx context.Context, limit *domain.RiskLimit) error {
	var existing domain.RiskLimit
	err := r.db.WithContext(ctx).Where("user_id = ? AND limit_type = ?", limit.UserID, limit.LimitType).First(&existing).Error
	if err == nil {
		limit.ID = existing.ID
		limit.CreatedAt = existing.CreatedAt
		return r.db.WithContext(ctx).Save(limit).Error
	}
	return r.db.WithContext(ctx).Create(limit).Error
}

func (r *riskRepository) GetLimitsByUserID(ctx context.Context, userID string) ([]*domain.RiskLimit, error) {
	var limits []*domain.RiskLimit
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&limits).Error
	return limits, err
}

func (r *riskRepository) GetLimitByUserIDAndType(ctx context.Context, userID, limitType string) (*domain.RiskLimit, error) {
	var limit domain.RiskLimit
	err := r.db.WithContext(ctx).Where("user_id = ? AND limit_type = ?", userID, limitType).First(&limit).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &limit, err
}

func (r *riskRepository) GetLimit(ctx context.Context, id string) (*domain.RiskLimit, error) {
	var limit domain.RiskLimit
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&limit).Error
	return &limit, err
}

// --- RiskAssessmentRepository 实现 ---

func (r *riskRepository) SaveAssessment(ctx context.Context, assessment *domain.RiskAssessment) error {
	return r.db.WithContext(ctx).Save(assessment).Error
}

func (r *riskRepository) GetAssessment(ctx context.Context, id string) (*domain.RiskAssessment, error) {
	var assessment domain.RiskAssessment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&assessment).Error
	return &assessment, err
}

func (r *riskRepository) GetLatestAssessmentByUser(ctx context.Context, userID string) (*domain.RiskAssessment, error) {
	var assessment domain.RiskAssessment
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").First(&assessment).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &assessment, err
}

// --- RiskMetricsRepository 实现 ---

func (r *riskRepository) SaveMetrics(ctx context.Context, metrics *domain.RiskMetrics) error {
	return r.db.WithContext(ctx).Save(metrics).Error
}

func (r *riskRepository) GetMetrics(ctx context.Context, userID string) (*domain.RiskMetrics, error) {
	var metrics domain.RiskMetrics
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&metrics).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &metrics, err
}

// --- RiskAlertRepository 实现 ---

func (r *riskRepository) SaveAlert(ctx context.Context, alert *domain.RiskAlert) error {
	return r.db.WithContext(ctx).Save(alert).Error
}

func (r *riskRepository) GetAlertsByUser(ctx context.Context, userID string, limit int) ([]*domain.RiskAlert, error) {
	var alerts []*domain.RiskAlert
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Limit(limit).Find(&alerts).Error
	return alerts, err
}

func (r *riskRepository) DeleteAlertByID(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.RiskAlert{}).Error
}

// --- CircuitBreakerRepository 实现 ---

func (r *riskRepository) SaveCircuitBreaker(ctx context.Context, cb *domain.CircuitBreaker) error {
	return r.db.WithContext(ctx).Save(cb).Error
}

func (r *riskRepository) GetCircuitBreakerByUserID(ctx context.Context, userID string) (*domain.CircuitBreaker, error) {
	var cb domain.CircuitBreaker
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&cb).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &cb, err
}
