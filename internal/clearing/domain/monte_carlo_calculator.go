//go:build clearing_experimental
// +build clearing_experimental

package domain

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algos/sim"
)

// MonteCarloMarginCalculator 蒙特卡洛保证金计算器
type MonteCarloMarginCalculator struct {
	volatilityRepo VolatilityRepository
	iterations     int
	confidence     float64
}

// NewMonteCarloMarginCalculator 创建 Monte Carlo 计算器
func NewMonteCarloMarginCalculator(volRepo VolatilityRepository) *MonteCarloMarginCalculator {
	return &MonteCarloMarginCalculator{
		volatilityRepo: volRepo,
		iterations:     10000,
		confidence:     0.99,
	}
}

// CalculateMargin 计算保证金
func (mc *MonteCarloMarginCalculator) CalculateMargin(ctx context.Context, portfolio *Portfolio) (*MarginResult, error) {
	var totalVaR float64
	var totalMarketValue float64
	var components []*MarginComponent

	for _, pos := range portfolio.Positions {
		if pos.Quantity == 0 {
			continue
		}

		qty := decimal.NewFromFloat(pos.Quantity)
		price := decimal.NewFromFloat(pos.Price)
		marketValue := qty.Mul(price).InexactFloat64()
		totalMarketValue += marketValue

		// 获取波动率
		vol, err := mc.volatilityRepo.GetVolatility(ctx, pos.Symbol, 1) // 1 day horizon
		if err != nil {
			// 如果获取失败，使用默认保守波动率 (e.g. 2%)
			vol = 0.02
		}

		// 设置 Geometric Brownian Motion 参数
		// Drift 设为 0 (风险中性/保守估计)
		drift := decimal.Zero
		volDec := decimal.NewFromFloat(vol)
		timeStep := decimal.NewFromFloat(1.0 / 252.0) // 日步长

		gbm := sim.NewGeometricBrownianMotion(price, drift, volDec, timeStep)
		mcSim := sim.NewMonteCarlo(gbm)

		// 计算 VaR (1步/1天，10000次模拟)
		varDec, err := mcSim.CalculateVaRMonteCarlo(1, mc.iterations, mc.confidence)
		if err != nil {
			return nil, fmt.Errorf("MC error for %s: %w", pos.Symbol, err)
		}

		// VaR 是百分比，转换为绝对金额
		// Position VaR = |MarketValue| * VaR_percent
		posVaR := math.Abs(marketValue * varDec.InexactFloat64())

		totalVaR += posVaR

		components = append(components, &MarginComponent{
			PositionID:        pos.ID, // 假设 Position 有 ID 字段
			Symbol:            pos.Symbol,
			MarketValue:       marketValue,
			RiskWeight:        varDec.InexactFloat64(), // 使用 VaR 作为风险权重
			MarginRequirement: posVaR,
		})
	}

	// 注意：此处简单累加 VaR (Correlation = 1)，忽略了多样化收益。
	// 完整实现应基于协方差矩阵模拟多资产联合路径。

	return &MarginResult{
		PortfolioID:      portfolio.ID,
		TotalMarketValue: totalMarketValue,
		NetMargin:        totalVaR,
		Components:       components,
		CalculatedAt:     time.Now(),
	}, nil
}
