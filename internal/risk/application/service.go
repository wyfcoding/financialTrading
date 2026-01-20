package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

type RiskApplicationService struct {
	repo domain.RiskLimitRepository
	// In future: OrderRepository, PositionRepository to check current usage
}

func NewRiskApplicationService(repo domain.RiskLimitRepository) *RiskApplicationService {
	return &RiskApplicationService{repo: repo}
}

func (s *RiskApplicationService) SetRiskLimit(ctx context.Context, userID string, maxOrderSize, maxDailyLoss float64) error {
	limit := &domain.RiskLimit{
		UserID:       userID,
		LimitType:    "ORDER_SIZE", // default or logic needed
		LimitValue:   decimal.NewFromFloat(maxOrderSize),
		CurrentValue: decimal.Zero,
	}
	return s.repo.Save(ctx, limit)
}

func (s *RiskApplicationService) CheckRisk(ctx context.Context, userID string, symbol string, quantity, price float64) (bool, string) {
	// 1. Get Limit
	limit, err := s.repo.GetByUserIDAndType(ctx, userID, "ORDER_SIZE")
	if err != nil {
		// If no limit found, decide default policy. Here: Reject or strict default.
		// Let's assume default rejection if no limits set for safety.
		return false, "No risk profile found"
	}
	if limit == nil {
		// No limit set for order size
		return true, ""
	}

	// 2. Check Order Size
	limitVal, _ := limit.LimitValue.Float64()
	if quantity > limitVal {
		return false, "Max order size exceeded"
	}

	return true, ""
}
