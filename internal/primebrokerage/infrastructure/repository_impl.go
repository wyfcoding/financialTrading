package infrastructure

import (
	"context"

	"github.com/wyfcoding/financialTrading/internal/primebrokerage/domain"
	"gorm.io/gorm"
)

// PrimeBrokerageRepositoryImpl 仓储实现
type PrimeBrokerageRepositoryImpl struct {
	db *gorm.DB
}

func NewPrimeBrokerageRepository(db *gorm.DB) domain.PrimeBrokerageRepository {
	return &PrimeBrokerageRepositoryImpl{db: db}
}

func (r *PrimeBrokerageRepositoryImpl) FindPoolBySymbol(ctx context.Context, symbol string) (*domain.SecurityPool, error) {
	var pool domain.SecurityPool
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&pool).Error; err != nil {
		return nil, err
	}
	return &pool, nil
}

func (r *PrimeBrokerageRepositoryImpl) UpdatePool(ctx context.Context, pool *domain.SecurityPool) error {
	return r.db.WithContext(ctx).Save(pool).Error
}

func (r *PrimeBrokerageRepositoryImpl) SaveLoan(ctx context.Context, loan *domain.SecurityLoan) error {
	return r.db.WithContext(ctx).Save(loan).Error
}

func (r *PrimeBrokerageRepositoryImpl) FindLoanByID(ctx context.Context, loanID string) (*domain.SecurityLoan, error) {
	var loan domain.SecurityLoan
	if err := r.db.WithContext(ctx).Where("loan_id = ?", loanID).First(&loan).Error; err != nil {
		return nil, err
	}
	return &loan, nil
}

func (r *PrimeBrokerageRepositoryImpl) ListActiveLoans(ctx context.Context, userID uint64) ([]*domain.SecurityLoan, error) {
	var loans []*domain.SecurityLoan
	if err := r.db.WithContext(ctx).Where("user_id = ? AND status = 'ACTIVE'", userID).Find(&loans).Error; err != nil {
		return nil, err
	}
	return loans, nil
}

func (r *PrimeBrokerageRepositoryImpl) SaveSeat(ctx context.Context, seat *domain.ClearingSeat) error {
	return r.db.WithContext(ctx).Save(seat).Error
}

func (r *PrimeBrokerageRepositoryImpl) ListSeats(ctx context.Context, exchange string) ([]*domain.ClearingSeat, error) {
	var seats []*domain.ClearingSeat
	if err := r.db.WithContext(ctx).Where("exchange_code = ?", exchange).Find(&seats).Error; err != nil {
		return nil, err
	}
	return seats, nil
}
