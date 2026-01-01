package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
)

// PricingManager 处理所有定价相关的写入操作（Commands）。
type PricingManager struct {
	marketDataClient domain.MarketDataClient
	pricingRepo      domain.PricingRepository
}

// NewPricingManager 构造函数。
func NewPricingManager(marketDataClient domain.MarketDataClient, pricingRepo domain.PricingRepository) *PricingManager {
	return &PricingManager{
		marketDataClient: marketDataClient,
		pricingRepo:      pricingRepo,
	}
}

// GetOptionPrice 计算期权价格 (Black-Scholes) 并保存结果
func (m *PricingManager) GetOptionPrice(ctx context.Context, contract domain.OptionContract, underlyingPrice decimal.Decimal, volatility, riskFreeRate float64) (decimal.Decimal, error) {
	timeToExpiry := float64(contract.ExpiryDate-time.Now().UnixMilli()) / 1000 / 24 / 3600 / 365
	s_val, _ := underlyingPrice.Float64()
	k_val, _ := contract.StrikePrice.Float64()

	result := domain.CalculateBlackScholes(contract.Type, domain.BlackScholesInput{
		S: s_val,
		K: k_val,
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
	}
	m.pricingRepo.Save(ctx, repoResult)

	return result.Price, nil
}
