package domain

import (
	"github.com/shopspring/decimal"
)

// CalculateMarginalVaR 基于蒙特卡洛模拟结果计算边际 VaR
// 核心思想：在总组合处于 VaR 尾部的那次模拟中，观察各资产的损失均值
func CalculateMarginalVaR(
	assets []PortfolioAsset,
	portfolioPnLs []float64,
	assetPnLs [][]float64, // [simulations][nAssets]
	confidenceLevel float64,
) map[string]float64 {
	nSims := len(portfolioPnLs)

	// 找到处于 VaR 临界点附近的模拟索引 (通常取尾部 1%-5% 的样本均值)
	tailIdx := int(float64(nSims) * (1 - confidenceLevel))
	if tailIdx < 1 {
		tailIdx = 1
	}

	marginalVaRs := make(map[string]float64)
	for i, asset := range assets {
		var tailSum float64
		// 对处于组合最差情况下的样本进行累加
		for s := 0; s < tailIdx; s++ {
			tailSum += assetPnLs[s][i]
		}
		// 边际风险贡献 (取反表示损失)
		marginalVaRs[asset.Symbol] = -tailSum / float64(tailIdx)
	}

	return marginalVaRs
}

// RunStressTests 运行压力测试场景 (支持多因子冲击)
func RunStressTests(assets []PortfolioAsset) map[string]decimal.Decimal {
	scenarios := map[string]struct {
		EquityShock float64
		VolShock    float64
		RatesShock  float64
	}{
		"Black_Monday_Replayed": {-0.22, 1.00, 0.05},
		"DotCom_Bubble_Burst":   {-0.15, 0.40, -0.02},
		"Covid_Liquidity_Gap":   {-0.12, 1.50, -0.10},
		"GFC_Credit_Crunch":     {-0.18, 0.80, 0.20},
	}

	results := make(map[string]decimal.Decimal)
	for name, scenario := range scenarios {
		var totalPnL float64
		for _, asset := range assets {
			posValue := asset.Position.Mul(asset.CurrentPrice).InexactFloat64()
			// 1. 直线价格冲击
			pnl := posValue * scenario.EquityShock
			// 2. 模拟波动率对持仓价值的影响 (假设部分持仓包含期权，VEGA 效应)
			// 此处为简化示例，实际应调用期权定价模型重估
			vegaImpact := posValue * 0.1 * scenario.VolShock

			totalPnL += pnl + vegaImpact
		}
		results[name] = decimal.NewFromFloat(totalPnL)
	}
	return results
}

// EstimatePortfolioGreeks 估算组合整体希腊字母 (基于风险暴露)
func EstimatePortfolioGreeks(assets []PortfolioAsset) map[string]PortfolioGreeks {
	greeks := make(map[string]PortfolioGreeks)
	for _, asset := range assets {
		// Delta: 对于 spot 来说，1 个单位持仓 = 1 个 Delta
		delta := asset.Position

		// 对于 Gamma, Vega, Theta:
		// 若是非线性产品，应通过 Finite Difference (扰动法) 计算:
		// Delta = (P(S+dS) - P(S-dS)) / (2*dS)
		// Gamma = (P(S+dS) - 2*P(S) + P(S-dS)) / (dS^2)

		// 默认实现假设为现货资产
		greeks[asset.Symbol] = PortfolioGreeks{
			Delta: delta,
			Gamma: decimal.Zero,
			Vega:  decimal.Zero,
			Theta: decimal.Zero,
		}
	}
	return greeks
}

// End of risk extensions
