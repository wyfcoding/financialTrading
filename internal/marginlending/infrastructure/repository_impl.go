package infrastructure

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marginlending/domain"
	"gorm.io/gorm"
)

type MarginRepositoryImpl struct {
	db *gorm.DB
}

func NewMarginRepository(db *gorm.DB) domain.MarginRepository {
	return &MarginRepositoryImpl{db: db}
}

func (r *MarginRepositoryImpl) SaveAccount(ctx context.Context, account *domain.MarginAccount) error {
	return r.db.WithContext(ctx).Save(account).Error
}

func (r *MarginRepositoryImpl) GetAccount(ctx context.Context, accountID string) (*domain.MarginAccount, error) {
	var account domain.MarginAccount
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *MarginRepositoryImpl) FindAccountByUserID(ctx context.Context, userID uint64) (*domain.MarginAccount, error) {
	var account domain.MarginAccount
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}
