// Package domain 包含风险管理服务的领域模型
package domain

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// MonteCarloInput 蒙特卡洛模拟输入参数
// 用于计算 VaR 和 ES
type MonteCarloInput struct {
	S          float64 // 当前价格
	Mu         float64 // 预期年化收益率
	Sigma      float64 // 年化波动率
	T          float64 // 时间跨度 (年)
	Iterations int     // 模拟次数 (例如 10000)
	Steps      int     // 时间步数 (例如 252)
}

// MonteCarloResult 蒙特卡洛模拟输出结果
type MonteCarloResult struct {
	VaR95 decimal.Decimal // 95% 置信度 VaR
	VaR99 decimal.Decimal // 99% 置信度 VaR
	ES95  decimal.Decimal // 95% 置信度预期亏损
	ES99  decimal.Decimal // 99% 置信度预期亏损
}

// CalculateVaR 使用蒙特卡洛模拟计算 VaR
func CalculateVaR(input MonteCarloInput) *MonteCarloResult {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	dt := input.T / float64(input.Steps)
	finalPrices := make([]float64, input.Iterations)

	for i := 0; i < input.Iterations; i++ {
		price := input.S
		for j := 0; j < input.Steps; j++ {
			// 几何布朗运动 (GBM): dS = S * (mu * dt + sigma * dW)
			// S(t+dt) = S(t) * exp((mu - 0.5 * sigma^2) * dt + sigma * sqrt(dt) * Z)
			z := r.NormFloat64()
			price *= math.Exp((input.Mu-0.5*input.Sigma*input.Sigma)*dt + input.Sigma*math.Sqrt(dt)*z)
		}
		finalPrices[i] = price
	}

	// 计算损益 (P&L)
	pnl := make([]float64, input.Iterations)
	for i, price := range finalPrices {
		pnl[i] = price - input.S
	}

	// 排序损益
	sort.Float64s(pnl)

	// 计算 VaR (取分位数)
	// VaR 通常表示为正数（损失金额）
	idx95 := int(float64(input.Iterations) * 0.05)
	idx99 := int(float64(input.Iterations) * 0.01)

	var95 := -pnl[idx95]
	var99 := -pnl[idx99]

	// 计算 Expected Shortfall (ES) / CVaR
	// ES 是超过 VaR 的损失的平均值
	var sumTail95, sumTail99 float64
	for i := 0; i < idx95; i++ {
		sumTail95 += pnl[i]
	}
	for i := 0; i < idx99; i++ {
		sumTail99 += pnl[i]
	}

	es95 := -sumTail95 / float64(idx95)
	es99 := -sumTail99 / float64(idx99)

	return &MonteCarloResult{
		VaR95: decimal.NewFromFloat(var95),
		VaR99: decimal.NewFromFloat(var99),
		ES95:  decimal.NewFromFloat(es95),
		ES99:  decimal.NewFromFloat(es99),
	}
}
