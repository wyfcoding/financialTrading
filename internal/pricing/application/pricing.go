package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
)

// PricingService 定价门面服务。
type PricingService struct {
	Command *PricingCommandService
	Query   *PricingQueryService
}

// NewPricingService 构造函数。
func NewPricingService(marketDataClient domain.MarketDataClient, repo domain.PricingRepository) *PricingService {
	return &PricingService{
		Command: NewPricingCommandService(marketDataClient, repo),
		Query:   NewPricingQueryService(repo),
	}
}

// --- Command Facade ---

func (s *PricingService) GetOptionPrice(ctx context.Context, contract domain.OptionContract, underlyingPrice decimal.Decimal, volatility, riskFreeRate float64) (decimal.Decimal, error) {
	return s.Command.GetOptionPrice(ctx, contract, underlyingPrice, volatility, riskFreeRate)
}

func (s *PricingService) OnQuoteReceived(ctx context.Context, symbol string, bid, ask float64, source string) error {
	return s.Command.OnQuoteReceived(ctx, symbol, bid, ask, source)
}

// --- Query Facade ---

func (s *PricingService) GetGreeks(ctx context.Context, contract domain.OptionContract, underlyingPrice decimal.Decimal, volatility, riskFreeRate float64) (*domain.Greeks, error) {
	return s.Query.GetGreeks(ctx, contract, underlyingPrice, volatility, riskFreeRate)
}

func (s *PricingService) GetLatestResult(ctx context.Context, symbol string) (*domain.PricingResult, error) {
	return s.Query.GetLatestResult(ctx, symbol)
}

func (s *PricingService) GetPrice(ctx context.Context, symbol string) (*PriceDTO, error) {
	return s.Query.GetPrice(ctx, symbol)
}

func (s *PricingService) ListPrices(ctx context.Context, symbols []string) ([]*PriceDTO, error) {
	return s.Query.ListPrices(ctx, symbols)
}
