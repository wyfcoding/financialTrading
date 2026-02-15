// 生成摘要：实现资金服务的 MySQL 仓储层，基于 GORM。
// 变更说明：从旧的 infrastructure 目录迁移至 persistence/mysql。

package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/funding/domain"
	"gorm.io/gorm"
)

// fundingRepository GORM 资金仓储实现
type fundingRepository struct {
	db *gorm.DB
}

// NewFundingRepository 创建资金仓储
func NewFundingRepository(db *gorm.DB) domain.FundingRepository {
	return &fundingRepository{db: db}
}

func (r *fundingRepository) SaveLoan(ctx context.Context, loan *domain.MarginLoan) error {
	return r.db.WithContext(ctx).Save(loan).Error
}

func (r *fundingRepository) GetLoan(ctx context.Context, loanID string) (*domain.MarginLoan, error) {
	var loan domain.MarginLoan
	err := r.db.WithContext(ctx).Where("loan_id = ?", loanID).First(&loan).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &loan, err
}

func (r *fundingRepository) ListLoans(ctx context.Context, userID string) ([]*domain.MarginLoan, error) {
	var loans []*domain.MarginLoan
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&loans).Error
	return loans, err
}

func (r *fundingRepository) SaveRate(ctx context.Context, rate *domain.FundingRate) error {
	return r.db.WithContext(ctx).Save(rate).Error
}

func (r *fundingRepository) GetLatestRate(ctx context.Context, symbol string) (*domain.FundingRate, error) {
	var rate domain.FundingRate
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp desc").First(&rate).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &rate, err
}
