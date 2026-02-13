package mysql

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marginlending/domain"
	"gorm.io/gorm"
)

type MarginAccountModel struct {
	gorm.Model
	AccountID       string          `gorm:"column:account_id;type:varchar(64);uniqueIndex;not null"`
	UserID          uint64          `gorm:"column:user_id;uniqueIndex;not null"`
	CollateralVal   decimal.Decimal `gorm:"column:collateral_val;type:decimal(20,8);not null"`
	BorrowedAmount  decimal.Decimal `gorm:"column:borrowed_amount;type:decimal(20,8);not null"`
	InterestAccrued decimal.Decimal `gorm:"column:interest_accrued;type:decimal(20,8);not null"`
	MarginRatio     decimal.Decimal `gorm:"column:margin_ratio;type:decimal(10,4);not null"`
	Status          string          `gorm:"column:status;type:varchar(20);not null"`
	LeverageLimit   int32           `gorm:"column:leverage_limit;not null"`
	LastInterestAt  time.Time       `gorm:"column:last_interest_at"`
}

func (MarginAccountModel) TableName() string { return "margin_accounts" }

type MarginRepo struct {
	db *gorm.DB
}

func NewMarginRepo(db *gorm.DB) domain.MarginRepository {
	return &MarginRepo{db: db}
}

func (r *MarginRepo) GetAccount(ctx context.Context, accountID string) (*domain.MarginAccount, error) {
	var model MarginAccountModel
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).First(&model).Error; err != nil {
		return nil, err
	}
	return toDomain(&model), nil
}

func (r *MarginRepo) FindAccountByUserID(ctx context.Context, userID uint64) (*domain.MarginAccount, error) {
	var model MarginAccountModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&model).Error; err != nil {
		return nil, err
	}
	return toDomain(&model), nil
}

func (r *MarginRepo) SaveAccount(ctx context.Context, acc *domain.MarginAccount) error {
	model := MarginAccountModel{
		AccountID:       acc.AccountID,
		UserID:          acc.UserID,
		CollateralVal:   acc.CollateralVal,
		BorrowedAmount:  acc.BorrowedAmount,
		InterestAccrued: acc.InterestAccrued,
		MarginRatio:     acc.MarginRatio,
		Status:          string(acc.Status),
		LeverageLimit:   acc.LeverageLimit,
		LastInterestAt:  acc.LastInterestAt,
	}

	var exist MarginAccountModel
	if err := r.db.WithContext(ctx).Where("account_id = ?", acc.AccountID).First(&exist).Error; err == nil {
		model.ID = exist.ID
		model.CreatedAt = exist.CreatedAt
	}

	return r.db.WithContext(ctx).Save(&model).Error
}

func toDomain(m *MarginAccountModel) *domain.MarginAccount {
	return &domain.MarginAccount{
		AccountID:       m.AccountID,
		UserID:          m.UserID,
		CollateralVal:   m.CollateralVal,
		BorrowedAmount:  m.BorrowedAmount,
		InterestAccrued: m.InterestAccrued,
		MarginRatio:     m.MarginRatio,
		Status:          domain.MarginStatus(m.Status),
		LeverageLimit:   m.LeverageLimit,
		LastInterestAt:  m.LastInterestAt,
	}
}
