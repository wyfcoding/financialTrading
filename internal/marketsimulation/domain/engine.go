package domain

import (
	"math"
	"math/rand"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algorithm/sim"
)

// PriceGenerator defines the interface for generating price updates
type PriceGenerator interface {
	Next(currentPrice float64, dt float64) float64
}

// GeometricBrownianMotion implements a GBM price process
type GeometricBrownianMotion struct {
	Drift      float64 // mu
	Volatility float64 // sigma
	Rand       *rand.Rand
}

func NewGBM(drift, volatility float64, seed int64) *GeometricBrownianMotion {
	return &GeometricBrownianMotion{
		Drift:      drift,
		Volatility: volatility,
		Rand:       rand.New(rand.NewSource(seed)),
	}
}

func (gbm *GeometricBrownianMotion) Next(currentPrice float64, dt float64) float64 {
	z := gbm.Rand.NormFloat64()
	return currentPrice * math.Exp((gbm.Drift-0.5*gbm.Volatility*gbm.Volatility)*dt+gbm.Volatility*math.Sqrt(dt)*z)
}

// HestonGenerator 赫斯顿模型状态包装
type HestonGenerator struct {
	impl   *sim.HestonModel
	curVol float64
}

func NewHestonGenerator(price, vol, kappa, theta, sigma, rho float64) *HestonGenerator {
	return &HestonGenerator{
		impl: sim.NewHestonModel(
			decimal.NewFromFloat(price),
			decimal.NewFromFloat(vol),
			decimal.NewFromFloat(kappa),
			decimal.NewFromFloat(theta),
			decimal.NewFromFloat(sigma),
			decimal.NewFromFloat(rho),
		),
		curVol: vol,
	}
}

func (h *HestonGenerator) Next(currentPrice float64, dt float64) float64 {
	// Heston Simulate 返回完整序列，此处我们仅需要单步演化逻辑.
	// 为保持接口一致，我们直接在 domain 侧实现轻量级单步演化或调用 pkg 优化版.
	res := h.impl.Simulate(1, decimal.NewFromFloat(dt))
	return res[1].InexactFloat64()
}

// JumpDiffusionGenerator 默顿跳跃扩散状态包装
type JumpDiffusionGenerator struct {
	impl *sim.MertonJumpDiffusion
}

func NewJumpDiffusionGenerator(price, drift, vol, lambda, jMu, jVol float64) *JumpDiffusionGenerator {
	return &JumpDiffusionGenerator{
		impl: &sim.MertonJumpDiffusion{
			InitialPrice: decimal.NewFromFloat(price),
			Drift:        decimal.NewFromFloat(drift),
			Volatility:   decimal.NewFromFloat(vol),
			JumpLambda:   decimal.NewFromFloat(lambda),
			JumpMu:       decimal.NewFromFloat(jMu),
			JumpSigma:    decimal.NewFromFloat(jVol),
		},
	}
}

func (j *JumpDiffusionGenerator) Next(currentPrice float64, dt float64) float64 {
	j.impl.InitialPrice = decimal.NewFromFloat(currentPrice)
	res := j.impl.Simulate(1, decimal.NewFromFloat(dt))
	return res[1].InexactFloat64()
}
