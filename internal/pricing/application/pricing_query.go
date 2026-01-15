package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
)

// PricingQuery 处理所有定价相关的查询操作（Queries）。
type PricingQuery struct {
	pricingRepo      domain.PricingRepository
	marketDataClient domain.MarketDataClient
}

// NewPricingQuery 构造函数。
func NewPricingQuery(marketDataClient domain.MarketDataClient, pricingRepo domain.PricingRepository) *PricingQuery {
	return &PricingQuery{
		marketDataClient: marketDataClient,
		pricingRepo:      pricingRepo,
	}
}

// GetGreeks 计算希腊字母
func (q *PricingQuery) GetGreeks(ctx context.Context, contract domain.OptionContract, underlyingPrice decimal.Decimal, volatility, riskFreeRate float64) (*domain.Greeks, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	timeToExpiry := float64(contract.ExpiryDate-time.Now().UnixMilli()) / 1000 / 24 / 3600 / 365
	if timeToExpiry < 0 {
		return &domain.Greeks{
			Delta: decimal.Zero,
			Gamma: decimal.Zero,
			Theta: decimal.Zero,
			Vega:  decimal.Zero,
			Rho:   decimal.Zero,
		}, nil
	}

	sVal, _ := underlyingPrice.Float64()
	kVal, _ := contract.StrikePrice.Float64()

	result := domain.CalculateBlackScholes(contract.Type, domain.BlackScholesInput{
		S: sVal,
		K: kVal,
		T: timeToExpiry,
		R: riskFreeRate,
		V: volatility,
	})

	return &domain.Greeks{
		Delta: result.Delta,
		Gamma: result.Gamma,
		Theta: result.Theta,
		Vega:  result.Vega,
		Rho:   result.Rho,
	}, nil
}

// GetLatestResult 获取最新定价结果
func (q *PricingQuery) GetLatestResult(ctx context.Context, symbol string) (*domain.PricingResult, error) {
	return q.pricingRepo.GetLatest(ctx, symbol)
}
