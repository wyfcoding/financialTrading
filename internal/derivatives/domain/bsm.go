// 变更说明：
// 1. 【核心定价】提供基础的 Black-Scholes-Merton 欧式期权（Call / Put）理论定价模型公式。
// 2. 【风险量化】支持希腊字母（Greeks: Delta, Gamma, Theta, Vega, Rho）的偏导数计算。
// 3. 【隐含波动率】Newton-Raphson 迭代求解 IV 引擎（用于期权做市与波动率曲面绘制）。
//go:build derivatives_experimental
// +build derivatives_experimental

package domain

import (
	"math"

	"github.com/shopspring/decimal"
)

// OptionType 期权类型
type OptionType string

const (
	OptionCall OptionType = "CALL" // 认购（看涨）
	OptionPut  OptionType = "PUT"  // 认沽（看跌）
)

// BSMParameters BSM模型参数
type BSMParameters struct {
	S float64 // 标的资产当前价格 (Spot Price)
	K float64 // 行权价 (Strike Price)
	T float64 // 到期时间 (Time to Maturity, 年化)
	R float64 // 无风险利率 (Risk-Free Rate, 年化连续复利)
	Q float64 // 连续股息率 (Dividend Yield)
	V float64 // 标的波动率 (Volatility, 年化)
}

// Greeks 期权希腊字母 (一阶/二阶风险敏感度)
type Greeks struct {
	Delta float64 `json:"delta"` // ∂V/∂S 对标的价格的一阶导
	Gamma float64 `json:"gamma"` // ∂²V/∂S² 对标的价格的二阶导
	Theta float64 `json:"theta"` // ∂V/∂T 对时间衰减的导数 (按天计)
	Vega  float64 `json:"vega"`  // ∂V/∂σ  对波动率的导数 (1% 波动率变动)
	Rho   float64 `json:"rho"`   // ∂V/∂R 对利率的导数 (1% 利率变动)
}

// normCDF 标准正态分布的累积分布函数 N(x)
func normCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

// normPDF 标准正态分布的概率密度函数 n(x)
func normPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2*math.Pi)
}

// calculateD1D2 计算 d1 和 d2
func calculateD1D2(p BSMParameters) (d1, d2 float64) {
	if p.T <= 0 || p.V <= 0 {
		return 0, 0
	}
	d1 = (math.Log(p.S/p.K) + (p.R-p.Q+0.5*p.V*p.V)*p.T) / (p.V * math.Sqrt(p.T))
	d2 = d1 - p.V*math.Sqrt(p.T)
	return d1, d2
}

// BSMPrice 计算欧式期权理论价格
func BSMPrice(optType OptionType, p BSMParameters) float64 {
	if p.T <= 0 {
		if optType == OptionCall {
			return math.Max(0, p.S-p.K)
		}
		return math.Max(0, p.K-p.S)
	}

	d1, d2 := calculateD1D2(p)
	discountedStrike := p.K * math.Exp(-p.R*p.T)
	dividendAdj := math.Exp(-p.Q * p.T)

	if optType == OptionCall {
		return p.S*dividendAdj*normCDF(d1) - discountedStrike*normCDF(d2)
	} else if optType == OptionPut {
		return discountedStrike*normCDF(-d2) - p.S*dividendAdj*normCDF(-d1)
	}
	return 0
}

// BSMGreeks 计算理论期权的风险因子
func BSMGreeks(optType OptionType, p BSMParameters) Greeks {
	var greeks Greeks

	if p.T <= 0 || p.V <= 0 {
		return greeks
	}

	d1, d2 := calculateD1D2(p)
	dividendAdj := math.Exp(-p.Q * p.T)
	discountedStrike := math.Exp(-p.R * p.T)
	sqrtT := math.Sqrt(p.T)
	n_d1 := normPDF(d1)

	// Gamma 和 Vega 对于 Call 和 Put 是相同的
	greeks.Gamma = (n_d1 * dividendAdj) / (p.S * p.V * sqrtT)
	greeks.Vega = p.S * n_d1 * sqrtT * dividendAdj * 0.01 // 转换为 1% 的波动

	if optType == OptionCall {
		// Call Delta
		greeks.Delta = dividendAdj * normCDF(d1)

		// Call Theta (除以 365 转为每日)
		term1 := -(p.S * p.V * n_d1 * dividendAdj) / (2 * sqrtT)
		term2 := p.R * p.K * discountedStrike * normCDF(d2)
		term3 := p.Q * p.S * dividendAdj * normCDF(d1)
		greeks.Theta = (term1 - term2 + term3) / 365.0

		// Call Rho (1% 变动)
		greeks.Rho = p.K * p.T * discountedStrike * normCDF(d2) * 0.01

	} else if optType == OptionPut {
		// Put Delta
		greeks.Delta = dividendAdj * (normCDF(d1) - 1.0)

		// Put Theta
		term1 := -(p.S * p.V * n_d1 * dividendAdj) / (2 * sqrtT)
		term2 := p.R * p.K * discountedStrike * normCDF(-d2)
		term3 := p.Q * p.S * dividendAdj * normCDF(-d1)
		greeks.Theta = (term1 + term2 - term3) / 365.0

		// Put Rho (1% 变动)
		greeks.Rho = -p.K * p.T * discountedStrike * normCDF(-d2) * 0.01
	}

	return greeks
}

// ImpliedVolatility Newton-Raphson 法反推隐含波动率 (IV)。
// targetPrice: 实际盘口市场价格 (通常为 Mid-Price)
func ImpliedVolatility(optType OptionType, p BSMParameters, targetPrice float64) (float64, error) {
	// 初始化猜测波动率 (0.2 ~ 20%)
	sigma := 0.2
	maxIterations := 100
	tolerance := 1e-5

	for i := 0; i < maxIterations; i++ {
		p.V = sigma

		// 计算基于猜测 IV 的理论期权价格
		price := BSMPrice(optType, p)
		diff := price - targetPrice

		if math.Abs(diff) < tolerance {
			return sigma, nil
		}

		// 根据 Vega 计算梯度 (Newton-Raphson 步长)
		greeks := BSMGreeks(optType, p)
		vega := greeks.Vega * 100 // 恢复为单位绝对导数 (因为之前*0.01处理为百分比了)

		if vega < 1e-8 { // 若对波动率极其不敏感，放弃迭代以防除零
			return sigma, errors.New("vega too close to zero")
		}

		sigma = sigma - (diff / vega)

		if sigma < 0.0001 { // 防止陷入负波动率
			sigma = 0.0001
		}
	}

	return sigma, errors.New("implied volatility calculation failed to converge")
}

// DerivativePosition 期权持仓。
type DerivativePosition struct {
	Symbol     string          `json:"symbol"`
	Type       OptionType      `json:"type"`
	Quantity   decimal.Decimal `json:"quantity"`   // 正数为买方 (Long)，负数为卖方 (Short)
	Multiplier int64           `json:"multiplier"` // 合约乘数，例如 100 股
	Metrics    Greeks          `json:"metrics"`
}
