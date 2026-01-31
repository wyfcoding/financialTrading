package application

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

// MarketMakingService 做市门面服务，整合 Manager 和 Query。
type MarketMakingService struct {
	manager *MarketMakingManager
	query   *MarketMakingQuery
}

// NewMarketMakingService 构造函数。
func NewMarketMakingService(
	repo domain.MarketMakingRepository,
	orderClient domain.OrderClient,
	marketDataClient domain.MarketDataClient,
	logger *slog.Logger,
) *MarketMakingService {
	return &MarketMakingService{
		manager: NewMarketMakingManager(repo, orderClient, marketDataClient, logger),
		query:   NewMarketMakingQuery(repo),
	}
}

// --- Manager (Writes) ---

func (s *MarketMakingService) SetStrategy(ctx context.Context, symbol string, spread, minOrderSize, maxOrderSize, maxPosition decimal.Decimal, status string) (string, error) {
	return s.manager.SetStrategy(ctx, symbol, spread, minOrderSize, maxOrderSize, maxPosition, status)
}

// --- Query (Reads) ---

func (s *MarketMakingService) GetStrategy(ctx context.Context, symbol string) (*domain.QuoteStrategy, error) {
	return s.query.GetStrategy(ctx, symbol)
}

func (s *MarketMakingService) GetPerformance(ctx context.Context, symbol string) (*domain.MarketMakingPerformance, error) {
	return s.query.GetPerformance(ctx, symbol)
}
