package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marginlending/domain"
)

type MarginAppService struct {
	repo   domain.MarginRepository
	logger *slog.Logger
}

func NewMarginAppService(repo domain.MarginRepository, logger *slog.Logger) *MarginAppService {
	return &MarginAppService{
		repo:   repo,
		logger: logger,
	}
}

// EvaluateMargin 评估保证金并返回是否合规
func (s *MarginAppService) EvaluateMargin(ctx context.Context, userID uint64, symbol string, quantity int64, price int64) (bool, float64, float64, error) {
	// Mock Calculation
	val := decimal.NewFromInt(price).Mul(decimal.NewFromInt(quantity))
	required := val.Mul(decimal.NewFromFloat(0.1)) // 10% Initial Margin

	// Check user available balance (Mock: assume user has infinite money for now or check account)
	// For production, this should call Account Service.

	return true, required.InexactFloat64(), 10.0, nil
}

// LockCollateral 锁定抵押品
func (s *MarginAppService) LockCollateral(ctx context.Context, userID uint64, asset string, amount int64) (string, bool, error) {
	acc, err := s.repo.FindAccountByUserID(ctx, userID)
	if err != nil {
		// Create if not exists ? Or error?
		// Create new account for simplified flow
		acc = &domain.MarginAccount{
			AccountID:      fmt.Sprintf("ACC-%d", userID),
			UserID:         userID,
			Status:         domain.MarginStatusNormal,
			LeverageLimit:  10,
			CollateralVal:  decimal.Zero,
			BorrowedAmount: decimal.Zero,
			LastInterestAt: time.Now(),
		}
	}

	// Lock Logic: Increase Collateral
	// In real world, asset must be transferred from Spot to Margin Wallet
	lockID := fmt.Sprintf("LOCK-%d-%d", userID, time.Now().UnixNano())

	val := decimal.NewFromInt(amount) // Assuming 1:1 value for simplicity, usually need oracle price
	acc.CollateralVal = acc.CollateralVal.Add(val)

	if err := s.repo.SaveAccount(ctx, acc); err != nil {
		return "", false, err
	}

	s.logger.InfoContext(ctx, "collateral locked", "user_id", userID, "amount", amount)
	return lockID, true, nil
}

// MarginCall 强平检查
func (s *MarginAppService) MarginCall(ctx context.Context, userID uint64) (float64, float64, bool, error) {
	acc, err := s.repo.FindAccountByUserID(ctx, userID)
	if err != nil {
		return 0, 0, false, fmt.Errorf("account not found")
	}

	// Re-calculate collateral value based on current oracle price (Mock)
	// Assume price dropped 0%

	equity := acc.CollateralVal.Sub(acc.BorrowedAmount)
	maintMargin := acc.BorrowedAmount.Mul(decimal.NewFromFloat(0.05)) // 5% MM

	isLiquidatable := equity.LessThan(maintMargin)

	return maintMargin.InexactFloat64(), equity.InexactFloat64(), isLiquidatable, nil
}
