package application

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

type MarketMakingApplicationService struct {
	repo domain.QuoteStrategyRepository
}

func NewMarketMakingApplicationService(repo domain.QuoteStrategyRepository) *MarketMakingApplicationService {
	return &MarketMakingApplicationService{repo: repo}
}

func (s *MarketMakingApplicationService) SetStrategy(ctx context.Context, cmd SetStrategyCommand) (string, error) {
	// Parse fields
	spread, _ := decimal.NewFromString(cmd.Spread)
	minSz, _ := decimal.NewFromString(cmd.MinOrderSize)
	maxSz, _ := decimal.NewFromString(cmd.MaxOrderSize)
	maxPos, _ := decimal.NewFromString(cmd.MaxPosition)

	strategy, err := s.repo.GetStrategyBySymbol(ctx, cmd.Symbol)
	if err != nil {
		return "", err
	}

	if strategy == nil {
		strategy = domain.NewQuoteStrategy(cmd.Symbol, spread, minSz, maxSz, maxPos)
	} else {
		strategy.UpdateConfig(spread, minSz, maxSz, maxPos)
	}

	if strings.ToUpper(cmd.Status) == "ACTIVE" {
		strategy.Activate()
	} else if strings.ToUpper(cmd.Status) == "PAUSED" {
		strategy.Pause()
	}

	if err := s.repo.SaveStrategy(ctx, strategy); err != nil {
		return "", err
	}
	return strategy.Symbol, nil // Using Symbol as ID for now
}

func (s *MarketMakingApplicationService) GetStrategy(ctx context.Context, symbol string) (*StrategyDTO, error) {
	strategy, err := s.repo.GetStrategyBySymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if strategy == nil {
		return nil, nil
	}

	return &StrategyDTO{
		ID:           strategy.Symbol,
		Symbol:       strategy.Symbol,
		Spread:       strategy.Spread.String(),
		MinOrderSize: strategy.MinOrderSize.String(),
		MaxOrderSize: strategy.MaxOrderSize.String(),
		MaxPosition:  strategy.MaxPosition.String(),
		Status:       string(strategy.Status),
		CreatedAt:    strategy.CreatedAt.UnixMilli(),
		UpdatedAt:    strategy.UpdatedAt.UnixMilli(),
	}, nil
}

func (s *MarketMakingApplicationService) GetPerformance(ctx context.Context, symbol string) (*PerformanceDTO, error) {
	// Mock implementation for now
	return &PerformanceDTO{
		Symbol:      symbol,
		TotalPnL:    1023.50,
		TotalVolume: 50000,
		TotalTrades: 125,
		SharpeRatio: 1.8,
	}, nil
}
