package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type riskRepository struct {
	db *gorm.DB
}

// NewRiskRepository 创建并返回一个新的 RiskRepository 实例。
func NewRiskRepository(db *gorm.DB) domain.RiskRepository {
	return &riskRepository{db: db}
}

// --- tx helpers ---

func (r *riskRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *riskRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *riskRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *riskRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

// --- Limit ---

func (r *riskRepository) SaveLimit(ctx context.Context, limit *domain.RiskLimit) error {
	if limit == nil {
		return nil
	}
	model := toRiskLimitModel(limit)
	db := r.getDB(ctx).WithContext(ctx)

	var existing RiskLimitModel
	query := db
	if limit.ID != "" {
		query = query.Where("id = ?", limit.ID)
	} else {
		query = query.Where("user_id = ? AND limit_type = ?", limit.UserID, limit.LimitType)
	}

	err := query.First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(model).Error
	}
	if err != nil {
		return err
	}

	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt
	return db.Save(model).Error
}

func (r *riskRepository) GetLimitsByUserID(ctx context.Context, userID string) ([]*domain.RiskLimit, error) {
	var models []*RiskLimitModel
	err := r.getDB(ctx).WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error
	if err != nil {
		return nil, err
	}
	limits := make([]*domain.RiskLimit, 0, len(models))
	for _, m := range models {
		limits = append(limits, toRiskLimit(m))
	}
	return limits, nil
}

func (r *riskRepository) GetLimitByUserIDAndType(ctx context.Context, userID, limitType string) (*domain.RiskLimit, error) {
	var model RiskLimitModel
	err := r.getDB(ctx).WithContext(ctx).Where("user_id = ? AND limit_type = ?", userID, limitType).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toRiskLimit(&model), err
}

func (r *riskRepository) GetLimit(ctx context.Context, id string) (*domain.RiskLimit, error) {
	var model RiskLimitModel
	err := r.getDB(ctx).WithContext(ctx).Where("id = ?", id).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toRiskLimit(&model), err
}

// --- Assessment ---

func (r *riskRepository) SaveAssessment(ctx context.Context, assessment *domain.RiskAssessment) error {
	if assessment == nil {
		return nil
	}
	model := toRiskAssessmentModel(assessment)
	db := r.getDB(ctx).WithContext(ctx)

	var existing RiskAssessmentModel
	err := db.Where("id = ?", assessment.ID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(model).Error
	}
	if err != nil {
		return err
	}

	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt
	return db.Save(model).Error
}

func (r *riskRepository) GetAssessment(ctx context.Context, id string) (*domain.RiskAssessment, error) {
	var model RiskAssessmentModel
	err := r.getDB(ctx).WithContext(ctx).Where("id = ?", id).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toRiskAssessment(&model), err
}

func (r *riskRepository) GetLatestAssessmentByUser(ctx context.Context, userID string) (*domain.RiskAssessment, error) {
	var model RiskAssessmentModel
	err := r.getDB(ctx).WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toRiskAssessment(&model), err
}

// --- Metrics ---

func (r *riskRepository) SaveMetrics(ctx context.Context, metrics *domain.RiskMetrics) error {
	if metrics == nil {
		return nil
	}
	model := toRiskMetricsModel(metrics)
	db := r.getDB(ctx).WithContext(ctx)

	var existing RiskMetricsModel
	err := db.Where("user_id = ?", metrics.UserID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(model).Error
	}
	if err != nil {
		return err
	}

	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt
	return db.Save(model).Error
}

func (r *riskRepository) GetMetrics(ctx context.Context, userID string) (*domain.RiskMetrics, error) {
	var model RiskMetricsModel
	err := r.getDB(ctx).WithContext(ctx).Where("user_id = ?", userID).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toRiskMetrics(&model), err
}

// --- Alert ---

func (r *riskRepository) SaveAlert(ctx context.Context, alert *domain.RiskAlert) error {
	if alert == nil {
		return nil
	}
	model := toRiskAlertModel(alert)
	db := r.getDB(ctx).WithContext(ctx)

	var existing RiskAlertModel
	err := db.Where("id = ?", alert.ID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(model).Error
	}
	if err != nil {
		return err
	}

	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt
	return db.Save(model).Error
}

func (r *riskRepository) GetAlertByID(ctx context.Context, id string) (*domain.RiskAlert, error) {
	var model RiskAlertModel
	err := r.getDB(ctx).WithContext(ctx).Where("id = ?", id).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toRiskAlert(&model), err
}

func (r *riskRepository) GetAlertsByUser(ctx context.Context, userID string, limit int) ([]*domain.RiskAlert, error) {
	var models []*RiskAlertModel
	err := r.getDB(ctx).WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&models).Error
	if err != nil {
		return nil, err
	}
	alerts := make([]*domain.RiskAlert, 0, len(models))
	for _, m := range models {
		alerts = append(alerts, toRiskAlert(m))
	}
	return alerts, nil
}

func (r *riskRepository) DeleteAlertByID(ctx context.Context, id string) error {
	return r.getDB(ctx).WithContext(ctx).Where("id = ?", id).Delete(&RiskAlertModel{}).Error
}

// --- CircuitBreaker ---

func (r *riskRepository) SaveCircuitBreaker(ctx context.Context, cb *domain.CircuitBreaker) error {
	if cb == nil {
		return nil
	}
	model := toCircuitBreakerModel(cb)
	db := r.getDB(ctx).WithContext(ctx)

	var existing CircuitBreakerModel
	err := db.Where("user_id = ?", cb.UserID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(model).Error
	}
	if err != nil {
		return err
	}

	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt
	return db.Save(model).Error
}

func (r *riskRepository) GetCircuitBreakerByUserID(ctx context.Context, userID string) (*domain.CircuitBreaker, error) {
	var model CircuitBreakerModel
	err := r.getDB(ctx).WithContext(ctx).Where("user_id = ?", userID).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toCircuitBreaker(&model), err
}

func (r *riskRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
