// 包 定价服务的用例逻辑
package application

import (
	"context"
	"math"
	"time"

	"github.com/wyfcoding/financialTrading/internal/pricing/domain"
	"github.com/wyfcoding/pkg/logging"
)

// PricingService 定价应用服务
// 负责期权定价和希腊字母计算 (基于 Black-Scholes 模型)
type PricingService struct {
	marketDataClient domain.MarketDataClient // 市场数据客户端
}

// NewPricingService 创建定价应用服务实例
// marketDataClient: 注入的市场数据客户端
func NewPricingService(marketDataClient domain.MarketDataClient) *PricingService {
	return &PricingService{
		marketDataClient: marketDataClient,
	}
}

// GetOptionPrice 计算期权价格 (Black-Scholes)
func (s *PricingService) GetOptionPrice(ctx context.Context, contract domain.OptionContract, underlyingPrice, volatility, riskFreeRate float64) (float64, error) {
	logging.Info(ctx, "Calculating option price",
		"symbol", contract.Symbol,
		"type", contract.Type,
		"strike_price", contract.StrikePrice,
		"expiry_date", contract.ExpiryDate,
		"underlying_price", underlyingPrice,
	)

	timeToExpiry := time.Until(contract.ExpiryDate).Hours() / 24 / 365
	if timeToExpiry < 0 {
		return 0, nil
	}

	d1 := (math.Log(underlyingPrice/contract.StrikePrice) + (riskFreeRate+0.5*volatility*volatility)*timeToExpiry) / (volatility * math.Sqrt(timeToExpiry))
	d2 := d1 - volatility*math.Sqrt(timeToExpiry)

	var price float64
	if contract.Type == domain.OptionTypeCall {
		price = underlyingPrice*normalCDF(d1) - contract.StrikePrice*math.Exp(-riskFreeRate*timeToExpiry)*normalCDF(d2)
	} else {
		price = contract.StrikePrice*math.Exp(-riskFreeRate*timeToExpiry)*normalCDF(-d2) - underlyingPrice*normalCDF(-d1)
	}

	return price, nil
}

// GetGreeks 计算希腊字母
func (s *PricingService) GetGreeks(ctx context.Context, contract domain.OptionContract, underlyingPrice, volatility, riskFreeRate float64) (*domain.Greeks, error) {
	logging.Info(ctx, "Calculating Greeks",
		"symbol", contract.Symbol,
		"type", contract.Type,
		"strike_price", contract.StrikePrice,
		"expiry_date", contract.ExpiryDate,
		"underlying_price", underlyingPrice,
	)

	timeToExpiry := time.Until(contract.ExpiryDate).Hours() / 24 / 365
	if timeToExpiry < 0 {
		return &domain.Greeks{}, nil
	}

	d1 := (math.Log(underlyingPrice/contract.StrikePrice) + (riskFreeRate+0.5*volatility*volatility)*timeToExpiry) / (volatility * math.Sqrt(timeToExpiry))
	d2 := d1 - volatility*math.Sqrt(timeToExpiry)

	greeks := &domain.Greeks{}

	// Delta: 标的价格变化导致的期权价格变化率
	if contract.Type == domain.OptionTypeCall {
		greeks.Delta = normalCDF(d1)
	} else {
		greeks.Delta = normalCDF(d1) - 1
	}

	// Gamma: Delta 随标的价格变化的速率
	greeks.Gamma = normalPDF(d1) / (underlyingPrice * volatility * math.Sqrt(timeToExpiry))

	// Vega: 波动率变化导致的期权价格变化 (每 1% 变化的敏感度)
	greeks.Vega = underlyingPrice * normalPDF(d1) * math.Sqrt(timeToExpiry) / 100

	// Theta: 时间流逝导致的期权价格变化 (时间损耗)
	if contract.Type == domain.OptionTypeCall {
		greeks.Theta = (-underlyingPrice*normalPDF(d1)*volatility/(2*math.Sqrt(timeToExpiry)) - riskFreeRate*contract.StrikePrice*math.Exp(-riskFreeRate*timeToExpiry)*normalCDF(d2)) / 365
	} else {
		greeks.Theta = (-underlyingPrice*normalPDF(d1)*volatility/(2*math.Sqrt(timeToExpiry)) + riskFreeRate*contract.StrikePrice*math.Exp(-riskFreeRate*timeToExpiry)*normalCDF(-d2)) / 365
	}

	// Rho: 利率变化导致的期权价格变化
	if contract.Type == domain.OptionTypeCall {
		greeks.Rho = contract.StrikePrice * timeToExpiry * math.Exp(-riskFreeRate*timeToExpiry) * normalCDF(d2) / 100
	} else {
		greeks.Rho = -contract.StrikePrice * timeToExpiry * math.Exp(-riskFreeRate*timeToExpiry) * normalCDF(-d2) / 100
	}

	return greeks, nil
}

// normalCDF 标准正态分布累积分布函数
func normalCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

// normalPDF 标准正态分布概率密度函数
func normalPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2*math.Pi)
}
