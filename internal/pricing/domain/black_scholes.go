package domain

import (
	"math"

	"github.com/shopspring/decimal"
)

// BlackScholesInput Black-Scholes 模型输入
type BlackScholesInput struct {
	S float64 // 标的资产价格
	K float64 // 执行价格
	T float64 // 到期时间 (年)
	R float64 // 无风险利率
	V float64 // 波动率
}

// BlackScholesResult Black-Scholes 模型输出
type BlackScholesResult struct {
	Price decimal.Decimal
	Delta decimal.Decimal
	Gamma decimal.Decimal
	Theta decimal.Decimal
	Vega  decimal.Decimal
	Rho   decimal.Decimal
}

// CalculateBlackScholes 计算 Black-Scholes 价格和 Greeks
func CalculateBlackScholes(optionType OptionType, input BlackScholesInput) *BlackScholesResult {
	d1 := (math.Log(input.S/input.K) + (input.R+0.5*input.V*input.V)*input.T) / (input.V * math.Sqrt(input.T))
	d2 := d1 - input.V*math.Sqrt(input.T)

	var price, delta, theta, rho float64
	gamma := math.Exp(-d1*d1/2) / (input.S * input.V * math.Sqrt(2*math.Pi*input.T))
	vega := input.S * math.Sqrt(input.T) * math.Exp(-d1*d1/2) / math.Sqrt(2*math.Pi)

	if optionType == OptionTypeCall {
		price = input.S*normCdf(d1) - input.K*math.Exp(-input.R*input.T)*normCdf(d2)
		delta = normCdf(d1)
		theta = (-input.S*normPdf(d1)*input.V/(2*math.Sqrt(input.T)) - input.R*input.K*math.Exp(-input.R*input.T)*normCdf(d2))
		rho = input.K * input.T * math.Exp(-input.R*input.T) * normCdf(d2)
	} else {
		price = input.K*math.Exp(-input.R*input.T)*normCdf(-d2) - input.S*normCdf(-d1)
		delta = normCdf(d1) - 1
		theta = (-input.S*normPdf(d1)*input.V/(2*math.Sqrt(input.T)) + input.R*input.K*math.Exp(-input.R*input.T)*normCdf(-d2))
		rho = -input.K * input.T * math.Exp(-input.R*input.T) * normCdf(-d2)
	}

	return &BlackScholesResult{
		Price: decimal.NewFromFloat(price),
		Delta: decimal.NewFromFloat(delta),
		Gamma: decimal.NewFromFloat(gamma),
		Theta: decimal.NewFromFloat(theta),
		Vega:  decimal.NewFromFloat(vega),
		Rho:   decimal.NewFromFloat(rho),
	}
}

// normCdf 标准正态分布累积分布函数
func normCdf(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

// normPdf 标准正态分布概率密度函数
func normPdf(x float64) float64 {
	return math.Exp(-x*x/2) / math.Sqrt(2*math.Pi)
}
