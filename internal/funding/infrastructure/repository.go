package infrastructure

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/funding/domain"
	"gorm.io/gorm"
)

type GormFundingRepository struct {
	db *gorm.DB
}

func NewGormFundingRepository(db *gorm.DB) *GormFundingRepository {
	return &GormFundingRepository{db: db}
}

func (r *GormFundingRepository) SaveLoan(ctx context.Context, loan *domain.MarginLoan) error {
	return r.db.WithContext(ctx).Save(loan).Error
}

func (r *GormFundingRepository) GetLoan(ctx context.Context, loanID string) (*domain.MarginLoan, error) {
	var loan domain.MarginLoan
	err := r.db.WithContext(ctx).Where("loan_id = ?", loanID).First(&loan).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &loan, err
}

func (r *GormFundingRepository) ListLoans(ctx context.Context, userID string) ([]*domain.MarginLoan, error) {
	var loans []*domain.MarginLoan
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&loans).Error
	return loans, err
}

func (r *GormFundingRepository) SaveRate(ctx context.Context, rate *domain.FundingRate) error {
	return r.db.WithContext(ctx).Save(rate).Error
}

func (r *GormFundingRepository) GetLatestRate(ctx context.Context, symbol string) (*domain.FundingRate, error) {
	var rate domain.FundingRate
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp desc").First(&rate).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &rate, err
}
