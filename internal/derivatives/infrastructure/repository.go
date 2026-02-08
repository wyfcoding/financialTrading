package infrastructure

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/derivatives/domain"
	"gorm.io/gorm"
)

type ContractRepository struct {
	db *gorm.DB
}

func NewContractRepository(db *gorm.DB) *ContractRepository {
	return &ContractRepository{db: db}
}

func (r *ContractRepository) Save(ctx context.Context, c *domain.Contract) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *ContractRepository) Get(ctx context.Context, id string) (*domain.Contract, error) {
	var c domain.Contract
	if err := r.db.WithContext(ctx).Where("contract_id = ?", id).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ContractRepository) List(ctx context.Context, underlying string, activeOnly bool) ([]domain.Contract, error) {
	var cts []domain.Contract
	tx := r.db.WithContext(ctx)

	if underlying != "" {
		tx = tx.Where("underlying = ?", underlying)
	}
	if activeOnly {
		tx = tx.Where("status = ?", domain.StatusTrading)
	}

	if err := tx.Find(&cts).Error; err != nil {
		return nil, err
	}
	return cts, nil
}
