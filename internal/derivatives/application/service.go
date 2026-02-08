package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/derivatives/domain"
)

type DerivativesService struct {
	repo domain.ContractRepository
}

func NewDerivativesService(repo domain.ContractRepository) *DerivativesService {
	return &DerivativesService{repo: repo}
}

func (s *DerivativesService) CreateContract(ctx context.Context, symbol, underlying, typeStr string, strike float64, expiry time.Time, mult float64) (string, error) {
	id := fmt.Sprintf("CON-%d", time.Now().UnixNano())

	cType := domain.TypeCall
	if typeStr == "PUT" {
		cType = domain.TypePut
	} else if typeStr == "FUTURE" {
		cType = domain.TypeFuture
	}

	contract := domain.NewContract(id, symbol, underlying, cType, decimal.NewFromFloat(strike), expiry, decimal.NewFromFloat(mult))

	if err := s.repo.Save(ctx, contract); err != nil {
		return "", err
	}
	return id, nil
}

func (s *DerivativesService) ExerciseContract(ctx context.Context, contractID, userID string, quantity float64) (string, float64, error) {
	c, err := s.repo.Get(ctx, contractID)
	if err != nil {
		return "", 0, err
	}

	if c.IsExpired() {
		return "", 0, fmt.Errorf("contract expired")
	}

	// In a real system, we would:
	// 1. Check user position
	// 2. Calculate PnL based on current spot price
	// 3. Create a settlement record

	// Mock PnL calculation
	// Assume Spot > Strike for Call
	pnl := 100.0 * quantity // Placeholder

	return "SETT-" + fmt.Sprintf("%d", time.Now().Unix()), pnl, nil
}

func (s *DerivativesService) ListContracts(ctx context.Context, underlying string, activeOnly bool) ([]domain.Contract, error) {
	return s.repo.List(ctx, underlying, activeOnly)
}

func (s *DerivativesService) GetContract(ctx context.Context, id string) (*domain.Contract, error) {
	return s.repo.Get(ctx, id)
}
