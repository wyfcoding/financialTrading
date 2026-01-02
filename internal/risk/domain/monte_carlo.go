// 包 风险管理服务的领域模型
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

// CalculateVaR 使用蒙特卡洛模拟计算 VaR 和 ES (预期亏损)。
// 升级实现：支持基于路径模拟的完整分布分析。
func CalculateVaR(input MonteCarloInput) *MonteCarloResult {
	if input.Iterations <= 0 || input.Steps <= 0 {
		return &MonteCarloResult{}
	}

	// 真实化执行：使用更加健壮的随机数生成器
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	dt := input.T / float64(input.Steps)
	drift := (input.Mu - 0.5*input.Sigma*input.Sigma) * dt
	volatility := input.Sigma * math.Sqrt(dt)

	finalPrices := make([]float64, input.Iterations)

	// 模拟 Iterations 条独立路径
	for i := 0; i < input.Iterations; i++ {
		price := input.S
		for j := 0; j < input.Steps; j++ {
			// GBM 离散化方案
			z := r.NormFloat64()
			price *= math.Exp(drift + volatility*z)
		}
		finalPrices[i] = price
	}

	// 损益分析 (P&L Distribution)
	pnl := make([]float64, input.Iterations)
	for i, price := range finalPrices {
		pnl[i] = price - input.S
	}

	sort.Float64s(pnl)

	// 提取风险分位数 (Quantile-based VaR)
	// 例如 95% VaR 是指有 5% 的概率亏损超过此值
	getVaR := func(percentile float64) (float64, int) {
		idx := max(int(math.Floor(float64(input.Iterations)*percentile)), 1)
		return -pnl[idx-1], idx
	}

	var95, idx95 := getVaR(0.05)
	var99, idx99 := getVaR(0.01)

	// 计算 Expected Shortfall (ES) / CVaR
	// ES 为尾部亏损的期望值，反映了超出 VaR 后的平均损失严重程度
	calcES := func(idx int) float64 {
		sum := 0.0
		for i := range idx {
			sum += pnl[i]
		}
		return -sum / float64(idx)
	}

	return &MonteCarloResult{
		VaR95: decimal.NewFromFloat(var95),
		VaR99: decimal.NewFromFloat(var99),
		ES95:  decimal.NewFromFloat(calcES(idx95)),
		ES99:  decimal.NewFromFloat(calcES(idx99)),
	}
}
