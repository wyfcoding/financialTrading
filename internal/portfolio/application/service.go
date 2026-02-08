package application

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/portfolio/domain"
)

type PortfolioService struct {
	repo domain.PortfolioRepository
}

func NewPortfolioService(repo domain.PortfolioRepository) *PortfolioService {
	return &PortfolioService{repo: repo}
}

// GetOverview returns real-time portfolio overview
// In a real system, this would query Position Service, Account Service, and Market Data Service
func (s *PortfolioService) GetOverview(ctx context.Context, userID, currency string) (float64, float64, float64, float64, error) {
	// Mock implementation
	// Real implementation should use gRPC clients to call other services
	totalEquity := 10500.00
	unrealizedPnL := 500.00
	realizedPnL := 120.00
	dailyPnLPct := 0.0125 // 1.25%

	return totalEquity, unrealizedPnL, realizedPnL, dailyPnLPct, nil
}

// GetPositions returns detailed positions
func (s *PortfolioService) GetPositions(ctx context.Context, userID string) ([]struct {
	Symbol        string
	Qty           float64
	AvgPrice      float64
	CurrentPrice  float64
	MarketValue   float64
	UnrealizedPnL float64
	PnLPct        float64
	Type          string
}, error) {
	// Mock implementation
	return []struct {
		Symbol        string
		Qty           float64
		AvgPrice      float64
		CurrentPrice  float64
		MarketValue   float64
		UnrealizedPnL float64
		PnLPct        float64
		Type          string
	}{
		{"BTC/USD", 0.1, 60000, 65000, 6500, 500, 0.0833, "SPOT"},
		{"ETH/USD", 1.5, 3000, 3100, 4650, 150, 0.0333, "SPOT"},
	}, nil
}

// GetPerformance returns historical performance
func (s *PortfolioService) GetPerformance(ctx context.Context, userID, timeframe string) ([]domain.PortfolioSnapshot, *domain.UserPerformance, error) {
	// Determine time range based on timeframe
	end := time.Now()
	start := end.AddDate(0, -1, 0) // Default 1M

	if timeframe == "1Y" {
		start = end.AddDate(-1, 0, 0)
	}

	history, err := s.repo.GetSnapshots(ctx, userID, start, end)
	if err != nil {
		return nil, nil, err
	}

	perf, err := s.repo.GetPerformance(ctx, userID)
	if err != nil {
		// return empty perf if not found
		perf = &domain.UserPerformance{}
	}

	return history, perf, nil
}
