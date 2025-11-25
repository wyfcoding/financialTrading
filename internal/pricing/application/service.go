package application

import (
	"context"
	"math"
	"time"

	"github.com/wyfcoding/financialTrading/internal/pricing/domain"
	"github.com/wyfcoding/financialTrading/pkg/logger"
)

// PricingService 应用服务
type PricingService struct {
	marketDataClient domain.MarketDataClient
}

// NewPricingService 创建应用服务实例
func NewPricingService(marketDataClient domain.MarketDataClient) *PricingService {
	return &PricingService{
		marketDataClient: marketDataClient,
	}
}

// GetOptionPrice 计算期权价格 (Black-Scholes)
func (s *PricingService) GetOptionPrice(ctx context.Context, contract domain.OptionContract, underlyingPrice, volatility, riskFreeRate float64) (float64, error) {
	logger.Info(ctx, "Calculating option price",
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
	logger.Info(ctx, "Calculating Greeks",
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

	// Delta
	if contract.Type == domain.OptionTypeCall {
		greeks.Delta = normalCDF(d1)
	} else {
		greeks.Delta = normalCDF(d1) - 1
	}

	// Gamma (Same for Call and Put)
	greeks.Gamma = normalPDF(d1) / (underlyingPrice * volatility * math.Sqrt(timeToExpiry))

	// Vega (Same for Call and Put)
	greeks.Vega = underlyingPrice * normalPDF(d1) * math.Sqrt(timeToExpiry) / 100 // Usually expressed per 1% change

	// Theta
	if contract.Type == domain.OptionTypeCall {
		greeks.Theta = (-underlyingPrice*normalPDF(d1)*volatility/(2*math.Sqrt(timeToExpiry)) - riskFreeRate*contract.StrikePrice*math.Exp(-riskFreeRate*timeToExpiry)*normalCDF(d2)) / 365
	} else {
		greeks.Theta = (-underlyingPrice*normalPDF(d1)*volatility/(2*math.Sqrt(timeToExpiry)) + riskFreeRate*contract.StrikePrice*math.Exp(-riskFreeRate*timeToExpiry)*normalCDF(-d2)) / 365
	}

	// Rho
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
