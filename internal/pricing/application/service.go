// Package application 提供了定价服务的用例逻辑
package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
	"github.com/wyfcoding/pkg/logging"
)

// PricingService 定价应用服务
// 负责期权定价和希腊字母计算 (基于 Black-Scholes 模型)
type PricingService struct {
	marketDataClient domain.MarketDataClient  // 市场数据客户端
	pricingRepo      domain.PricingRepository // 定价仓储接口
}

// NewPricingService 创建定价应用服务实例
func NewPricingService(marketDataClient domain.MarketDataClient, pricingRepo domain.PricingRepository) *PricingService {
	return &PricingService{
		marketDataClient: marketDataClient,
		pricingRepo:      pricingRepo,
	}
}

// GetOptionPrice 计算期权价格 (Black-Scholes)
func (s *PricingService) GetOptionPrice(ctx context.Context, contract domain.OptionContract, underlyingPrice decimal.Decimal, volatility, riskFreeRate float64) (decimal.Decimal, error) {
	logging.Info(ctx, "Calculating option price",
		"symbol", contract.Symbol,
		"type", contract.Type,
		"strike_price", contract.StrikePrice.String(),
		"expiry_date", contract.ExpiryDate,
		"underlying_price", underlyingPrice.String(),
	)

	timeToExpiry := float64(contract.ExpiryDate-time.Now().UnixMilli()) / 1000 / 24 / 3600 / 365
	s_val, ok1 := underlyingPrice.Float64()
	k_val, ok2 := contract.StrikePrice.Float64()
	if !ok1 || !ok2 {
		logging.Warn(ctx, "Precision loss during decimal to float64 conversion", "symbol", contract.Symbol)
	}

	result := domain.CalculateBlackScholes(contract.Type, domain.BlackScholesInput{
		S: s_val,
		K: k_val,
		T: timeToExpiry,
		R: riskFreeRate,
		V: volatility,
	})

	// 保存计算结果
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
	if err := s.pricingRepo.Save(ctx, repoResult); err != nil {
		logging.Warn(ctx, "Failed to save pricing results", "error", err)
	}

	return result.Price, nil
}

// GetGreeks 计算希腊字母
func (s *PricingService) GetGreeks(ctx context.Context, contract domain.OptionContract, underlyingPrice decimal.Decimal, volatility, riskFreeRate float64) (*domain.Greeks, error) {
	logging.Info(ctx, "Calculating Greeks",
		"symbol", contract.Symbol,
		"type", contract.Type,
		"strike_price", contract.StrikePrice.String(),
		"expiry_date", contract.ExpiryDate,
		"underlying_price", underlyingPrice.String(),
	)

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

	s_val, ok1 := underlyingPrice.Float64()
	k_val, ok2 := contract.StrikePrice.Float64()
	if !ok1 || !ok2 {
		logging.Warn(ctx, "Precision loss during decimal to float64 conversion for Greeks", "symbol", contract.Symbol)
	}

	result := domain.CalculateBlackScholes(contract.Type, domain.BlackScholesInput{
		S: s_val,
		K: k_val,
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
