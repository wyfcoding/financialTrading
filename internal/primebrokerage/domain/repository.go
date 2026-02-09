package domain

import (
	"context"
)

// PrimeBrokerageRepository 主经纪商仓储接口
type PrimeBrokerageRepository interface {
	FindPoolBySymbol(ctx context.Context, symbol string) (*SecurityPool, error)
	UpdatePool(ctx context.Context, pool *SecurityPool) error
	SaveLoan(ctx context.Context, loan *SecurityLoan) error
	FindLoanByID(ctx context.Context, loanID string) (*SecurityLoan, error)
	ListActiveLoans(ctx context.Context, userID uint64) ([]*SecurityLoan, error)
	SaveSeat(ctx context.Context, seat *ClearingSeat) error
	ListSeats(ctx context.Context, exchange string) ([]*ClearingSeat, error)
}

// SecurityPoolService 借券库领域服务
type SecurityPoolService struct {
	repo PrimeBrokerageRepository
}

func NewSecurityPoolService(repo PrimeBrokerageRepository) *SecurityPoolService {
	return &SecurityPoolService{repo: repo}
}
