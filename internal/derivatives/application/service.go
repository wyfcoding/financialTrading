package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/derivatives/domain"
)

type DerivativeRepo interface {
	Save(ctx context.Context, c *domain.Contract) error
	Get(ctx context.Context, id string) (*domain.Contract, error)
	List(ctx context.Context, underlying, cType string, activeOnly bool) ([]*domain.Contract, error)
}

type DerivativesAppService struct {
	repo    DerivativeRepo
	pricing domain.PricingModel
	logger  *slog.Logger
}

func NewDerivativesAppService(repo DerivativeRepo, pricing domain.PricingModel, logger *slog.Logger) *DerivativesAppService {
	return &DerivativesAppService{
		repo:    repo,
		pricing: pricing,
		logger:  logger,
	}
}

// CreateContract 创建新合约
func (s *DerivativesAppService) CreateContract(ctx context.Context, symbol, underlying, cType string, strike float64, expiry time.Time, mult float64) (string, error) {
	contractID := fmt.Sprintf("CON-%s-%d", symbol, time.Now().UnixNano())

	contract := domain.NewContract(
		contractID,
		symbol,
		underlying,
		domain.ContractType(cType),
		decimal.NewFromFloat(strike),
		expiry,
		decimal.NewFromFloat(mult),
	)

	if err := s.repo.Save(ctx, contract); err != nil {
		return "", fmt.Errorf("failed to save contract: %w", err)
	}

	s.logger.InfoContext(ctx, "contract created", "id", contractID, "symbol", symbol)
	return contractID, nil
}

func (s *DerivativesAppService) GetContract(ctx context.Context, id string) (*domain.Contract, error) {
	return s.repo.Get(ctx, id)
}

func (s *DerivativesAppService) ListContracts(ctx context.Context, underlying, cType string, activeOnly bool) ([]*domain.Contract, error) {
	return s.repo.List(ctx, underlying, cType, activeOnly)
}

// ExerciseContract 行权 (Mock)
func (s *DerivativesAppService) ExerciseContract(ctx context.Context, id, userID string, qty float64) (bool, string, float64, error) {
	contract, err := s.repo.Get(ctx, id)
	if err != nil {
		return false, "", 0, err
	}

	if contract.IsExpired() {
		return false, "", 0, fmt.Errorf("contract expired")
	}

	// Mock Settlement Logic
	pnl := 100.0 * qty // Dummy PnL
	settlementID := fmt.Sprintf("SET-%s-%s", id, userID)

	s.logger.InfoContext(ctx, "contract exercised", "id", id, "user", userID, "pnl", pnl)

	return true, settlementID, pnl, nil
}
