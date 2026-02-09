// 变更说明：新增希腊字母 (Greeks) 计算逻辑，支持期权风险对冲分析。
// 假设：采用 Black-Scholes 模型简化实现，无风险利率默认 3%，波动率年化。
package domain

import (
	"math"

	"github.com/shopspring/decimal"
)

// GreeksResult 希腊字母计算结果
type GreeksResult struct {
	Delta decimal.Decimal
	Gamma decimal.Decimal
	Vega  decimal.Decimal
	Theta decimal.Decimal
	Rho   decimal.Decimal
}

// OptionGreeksCalculator 期权定价与风控计算器
type OptionGreeksCalculator struct {
	InterestRate float64 // 无风险利率 (e.g., 0.03)
}

func NewOptionGreeksCalculator(r float64) *OptionGreeksCalculator {
	return &OptionGreeksCalculator{InterestRate: r}
}

// CalculateGreeks 计算期权希腊字母
func (c *OptionGreeksCalculator) CalculateGreeks(
	spot, strike, vol, expiry float64, // expiry in years
	isCall bool,
) GreeksResult {
	if expiry <= 0 {
		return GreeksResult{}
	}

	d1 := (math.Log(spot/strike) + (c.InterestRate+0.5*vol*vol)*expiry) / (vol * math.Sqrt(expiry))
	d2 := d1 - vol*math.Sqrt(expiry)

	// 计算 N(d1), N(d2) 和概率密度 n(d1)
	nd1 := c.normCDF(d1)
	nd2 := c.normCDF(d2)
	np1 := c.normPDF(d1)

	var delta, gamma, vega, theta, rho float64

	// Delta
	if isCall {
		delta = nd1
	} else {
		delta = nd1 - 1
	}

	// Gamma
	gamma = np1 / (spot * vol * math.Sqrt(expiry))

	// Vega
	vega = spot * math.Sqrt(expiry) * np1 / 100 // 通常表示波动率变化 1% 的影响

	// Theta (简化的，单位为天)
	term1 := -(spot * np1 * vol) / (2 * math.Sqrt(expiry))
	if isCall {
		term2 := c.InterestRate * strike * math.Exp(-c.InterestRate*expiry) * nd2
		theta = (term1 - term2) / 365
	} else {
		term2 := c.InterestRate * strike * math.Exp(-c.InterestRate*expiry) * c.normCDF(-d2)
		theta = (term1 + term2) / 365
	}

	// Rho
	if isCall {
		rho = strike * expiry * math.Exp(-c.InterestRate*expiry) * nd2 / 100
	} else {
		rho = -strike * expiry * math.Exp(-c.InterestRate*expiry) * c.normCDF(-d2) / 100
	}

	return GreeksResult{
		Delta: decimal.NewFromFloat(delta),
		Gamma: decimal.NewFromFloat(gamma),
		Vega:  decimal.NewFromFloat(vega),
		Theta: decimal.NewFromFloat(theta),
		Rho:   decimal.NewFromFloat(rho),
	}
}

// normCDF 标准正态分布累积分布函数 (CDF) 的近似实现
func (c *OptionGreeksCalculator) normCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt(2)))
}

// normPDF 标准正态分布概率密度函数 (PDF)
func (c *OptionGreeksCalculator) normPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2*math.Pi)
}
