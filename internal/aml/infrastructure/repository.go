package infrastructure

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/aml/domain"
	"gorm.io/gorm"
)

type GormAMLRepository struct {
	db *gorm.DB
}

func NewGormAMLRepository(db *gorm.DB) *GormAMLRepository {
	return &GormAMLRepository{db: db}
}

func (r *GormAMLRepository) SaveAlert(ctx context.Context, alert *domain.AMLAlert) error {
	return r.db.WithContext(ctx).Save(alert).Error
}

func (r *GormAMLRepository) GetAlert(ctx context.Context, id string) (*domain.AMLAlert, error) {
	var alert domain.AMLAlert
	err := r.db.WithContext(ctx).Where("alert_id = ?", id).First(&alert).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &alert, err
}

func (r *GormAMLRepository) ListAlertsByStatus(ctx context.Context, status string) ([]*domain.AMLAlert, error) {
	var alerts []*domain.AMLAlert
	query := r.db.WithContext(ctx)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&alerts).Error
	return alerts, err
}

func (r *GormAMLRepository) SaveRiskScore(ctx context.Context, score *domain.UserRiskScore) error {
	return r.db.WithContext(ctx).Save(score).Error
}

func (r *GormAMLRepository) GetRiskScore(ctx context.Context, userID string) (*domain.UserRiskScore, error) {
	var score domain.UserRiskScore
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&score).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &score, err
}
