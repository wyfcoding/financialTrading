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
