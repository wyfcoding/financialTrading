package domain

import (
	"math"
)

// PricingModel 定价模型接口
type PricingModel interface {
	CalculatePrice(contract *Contract, spotPrice float64, riskFreeRate float64, volatility float64, timeToExpiry float64) float64
	CalculateGreeks(contract *Contract, spotPrice float64, riskFreeRate float64, volatility float64, timeToExpiry float64) Greeks
}

type Greeks struct {
	Delta float64
	Gamma float64
	Theta float64
	Vega  float64
	Rho   float64
}

type BlackScholesModel struct{}

func NewBlackScholesModel() *BlackScholesModel {
	return &BlackScholesModel{}
}

// CalculatePrice 计算期权理论价格
func (bs *BlackScholesModel) CalculatePrice(c *Contract, S, r, sigma, T float64) float64 {
	K, _ := c.StrikePrice.Float64()

	if c.Type == TypeFuture {
		// Future Price = S * e^(rT) (Simplified cost of carry)
		return S * math.Exp(r*T)
	}

	d1 := (math.Log(S/K) + (r+0.5*sigma*sigma)*T) / (sigma * math.Sqrt(T))
	d2 := d1 - sigma*math.Sqrt(T)

	if c.Type == TypeCall {
		return S*normCDF(d1) - K*math.Exp(-r*T)*normCDF(d2)
	} else if c.Type == TypePut {
		return K*math.Exp(-r*T)*normCDF(-d2) - S*normCDF(-d1)
	}
	return 0
}

func (bs *BlackScholesModel) CalculateGreeks(c *Contract, S, r, sigma, T float64) Greeks {
	K, _ := c.StrikePrice.Float64()
	d1 := (math.Log(S/K) + (r+0.5*sigma*sigma)*T) / (sigma * math.Sqrt(T))
	// d2 := d1 - sigma*math.Sqrt(T)

	var greeks Greeks
	pdfD1 := normPDF(d1)

	if c.Type == TypeCall {
		greeks.Delta = normCDF(d1)
		greeks.Rho = K * T * math.Exp(-r*T) * normCDF(d1-sigma*math.Sqrt(T))
	} else if c.Type == TypePut {
		greeks.Delta = normCDF(d1) - 1
		greeks.Rho = -K * T * math.Exp(-r*T) * normCDF(-(d1 - sigma*math.Sqrt(T)))
	}

	greeks.Gamma = pdfD1 / (S * sigma * math.Sqrt(T))
	greeks.Vega = S * pdfD1 * math.Sqrt(T) / 100
	greeks.Theta = -(S*pdfD1*sigma)/(2*math.Sqrt(T)) - r*K*math.Exp(-r*T)*normCDF(d1-sigma*math.Sqrt(T))

	return greeks
}

func normCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

func normPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2*math.Pi)
}
