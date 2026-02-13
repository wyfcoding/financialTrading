package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/derivatives/domain"
	"gorm.io/gorm"
)

type DerivativeRepo struct {
	db *gorm.DB
}

func NewDerivativeRepo(db *gorm.DB) *DerivativeRepo {
	return &DerivativeRepo{db: db}
}

// ContractModel matches domain.Contract but we use domain struct directly as it has gorm tags
// Or we can alias it. domain.Contract already has gorm.Model.
// So we can use it directly.

func (r *DerivativeRepo) Save(ctx context.Context, c *domain.Contract) error {
	// Check exist for ID to preserve ID/Created
	var exist domain.Contract
	if err := r.db.WithContext(ctx).Where("contract_id = ?", c.ContractID).First(&exist).Error; err == nil {
		c.ID = exist.ID
		c.CreatedAt = exist.CreatedAt
	}
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *DerivativeRepo) Get(ctx context.Context, id string) (*domain.Contract, error) {
	var c domain.Contract
	if err := r.db.WithContext(ctx).Where("contract_id = ?", id).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *DerivativeRepo) List(ctx context.Context, underlying string, cType string, activeOnly bool) ([]*domain.Contract, error) {
	var contracts []*domain.Contract
	query := r.db.WithContext(ctx)

	if underlying != "" {
		query = query.Where("underlying = ?", underlying)
	}
	if cType != "" {
		query = query.Where("type = ?", cType)
	}
	if activeOnly {
		query = query.Where("status = ?", domain.StatusTrading)
	}

	err := query.Find(&contracts).Error
	return contracts, err
}
