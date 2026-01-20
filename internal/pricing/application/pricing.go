package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
)

// PricingService 定价门面服务，整合 Manager 和 Query。
type PricingService struct {
	manager *PricingManager
	query   *PricingQuery
}

// NewPricingService 构造函数。
func NewPricingService(marketDataClient domain.MarketDataClient, pricingRepo domain.PricingRepository, priceRepo domain.PriceRepository) *PricingService {
	return &PricingService{
		manager: NewPricingManager(marketDataClient, pricingRepo, priceRepo),
		query:   NewPricingQuery(marketDataClient, pricingRepo, priceRepo),
	}
}

// --- Manager (Writes) ---

func (s *PricingService) GetOptionPrice(ctx context.Context, contract domain.OptionContract, underlyingPrice decimal.Decimal, volatility, riskFreeRate float64) (decimal.Decimal, error) {
	return s.manager.GetOptionPrice(ctx, contract, underlyingPrice, volatility, riskFreeRate)
}

func (s *PricingService) OnQuoteReceived(ctx context.Context, symbol string, bid, ask float64, source string) error {
	return s.manager.OnQuoteReceived(ctx, symbol, bid, ask, source)
}

// --- Query (Reads) ---

func (s *PricingService) GetGreeks(ctx context.Context, contract domain.OptionContract, underlyingPrice decimal.Decimal, volatility, riskFreeRate float64) (*domain.Greeks, error) {
	return s.query.GetGreeks(ctx, contract, underlyingPrice, volatility, riskFreeRate)
}

func (s *PricingService) GetLatestResult(ctx context.Context, symbol string) (*domain.PricingResult, error) {
	return s.query.GetLatestResult(ctx, symbol)
}

func (s *PricingService) GetPrice(ctx context.Context, symbol string) (*PriceDTO, error) {
	return s.query.GetPrice(ctx, symbol)
}

func (s *PricingService) ListPrices(ctx context.Context, symbols []string) ([]*PriceDTO, error) {
	return s.query.ListPrices(ctx, symbols)
}
