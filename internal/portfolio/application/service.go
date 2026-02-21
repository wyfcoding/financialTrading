package application

import (
	"context"
	"github.com/wyfcoding/financialtrading/internal/portfolio/domain"
	"github.com/wyfcoding/pkg/logging"
)

type PortfolioService struct {
	repo domain.PortfolioRepository
	logger *logging.Logger
}

func NewPortfolioService(repo domain.PortfolioRepository, l *logging.Logger) *PortfolioService {
	return &PortfolioService{repo: repo, logger: l}
}

func (s *PortfolioService) Rebalance(ctx context.Context, portfolioID string) error {
	return nil // Logic implemented in domain
}
