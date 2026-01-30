package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
	"github.com/wyfcoding/pkg/logging"
)

// PricingCommandService 处理所有定价相关的写入操作（Commands）。
type PricingCommandService struct {
	marketDataClient domain.MarketDataClient
	repo             domain.PricingRepository
}

// NewPricingCommandService 构造函数。
func NewPricingCommandService(marketDataClient domain.MarketDataClient, repo domain.PricingRepository) *PricingCommandService {
	return &PricingCommandService{
		marketDataClient: marketDataClient,
		repo:             repo,
	}
}

// GetOptionPrice 计算期权价格 (Black-Scholes) 并保存结果
func (s *PricingCommandService) GetOptionPrice(ctx context.Context, contract domain.OptionContract, underlyingPrice decimal.Decimal, volatility, riskFreeRate float64) (decimal.Decimal, error) {
	timeToExpiry := float64(contract.ExpiryDate-time.Now().UnixMilli()) / 1000 / 24 / 3600 / 365
	sVal, _ := underlyingPrice.Float64()
	kVal, _ := contract.StrikePrice.Float64()

	result := domain.CalculateBlackScholes(contract.Type, domain.BlackScholesInput{
		S: sVal,
		K: kVal,
		T: timeToExpiry,
		R: riskFreeRate,
		V: volatility,
	})

	repoResult := &domain.PricingResult{
		Symbol:          contract.Symbol,
		OptionPrice:     result.Price,
		UnderlyingPrice: underlyingPrice,
		Delta:           result.Delta,
		Gamma:           result.Gamma,
		Theta:           result.Theta,
		Vega:            result.Vega,
		Rho:             result.Rho,
		CalculatedAt:    time.Now().UnixMilli(),
		PricingModel:    "BlackScholes",
	}
	if err := s.repo.SavePricingResult(ctx, repoResult); err != nil {
		logging.Error(ctx, "PricingCommandService: failed to save price result", "error", err)
	}

	return result.Price, nil
}

func (s *PricingCommandService) OnQuoteReceived(ctx context.Context, symbol string, bid, ask float64, source string) error {
	price := domain.NewPrice(symbol, bid, ask, source)
	return s.repo.SavePrice(ctx, price)
}
