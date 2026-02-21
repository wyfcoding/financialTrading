// 变更说明：
// 从 pkg/algos/sim/simulation.go 迁移。
// 实现了市场模拟算法，包括 GBM, 蒙特卡洛，Heston 模型等。
package domain

import (
	"encoding/binary"
	"math"
	"runtime"
	"sync"
	"time"

	crypto_rand "crypto/rand"

	"github.com/shopspring/decimal"
)

// GeometricBrownianMotion 几何布朗运动模拟.
type GeometricBrownianMotion struct {
	initialPrice decimal.Decimal
	drift        decimal.Decimal
	volatility   decimal.Decimal
	timeStep     decimal.Decimal
}

func NewGeometricBrownianMotion(initialPrice, drift, volatility, timeStep decimal.Decimal) *GeometricBrownianMotion {
	return &GeometricBrownianMotion{
		initialPrice: initialPrice,
		drift:        drift,
		volatility:   volatility,
		timeStep:     timeStep,
	}
}

func cryptoNormFloat64() float64 {
	var b [16]byte
	if _, err := crypto_rand.Read(b[:]); err != nil {
		ts := time.Now().UnixNano()
		binary.LittleEndian.PutUint64(b[:8], uint64(ts))
		binary.LittleEndian.PutUint64(b[8:], uint64(ts))
	}
	u1 := float64(binary.LittleEndian.Uint64(b[:8]))/float64(math.MaxUint64) + 1e-10
	u2 := float64(binary.LittleEndian.Uint64(b[8:])) / float64(math.MaxUint64)
	return math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
}

func (gbm *GeometricBrownianMotion) Simulate(steps int) []decimal.Decimal {
	prices := make([]decimal.Decimal, steps+1)
	prices[0] = gbm.initialPrice

	driftFloat := gbm.drift.InexactFloat64()
	volatilityFloat := gbm.volatility.InexactFloat64()
	timeStepFloat := gbm.timeStep.InexactFloat64()

	driftTerm := (driftFloat - 0.5*volatilityFloat*volatilityFloat) * timeStepFloat
	volTerm := volatilityFloat * math.Sqrt(timeStepFloat)

	for i := 1; i <= steps; i++ {
		z := cryptoNormFloat64()
		currentPrice := prices[i-1].InexactFloat64()
		exponent := driftTerm + volTerm*z
		newPrice := currentPrice * math.Exp(exponent)
		prices[i] = decimal.NewFromFloat(newPrice)
	}

	return prices
}

func (gbm *GeometricBrownianMotion) SimulateMultiplePaths(steps, paths int) [][]decimal.Decimal {
	allPaths := make([][]decimal.Decimal, paths)
	numWorkers := runtime.GOMAXPROCS(0)
	var wg sync.WaitGroup
	wg.Add(paths)
	sem := make(chan struct{}, numWorkers)

	for i := range paths {
		go func(pathIdx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			allPaths[pathIdx] = gbm.Simulate(steps)
		}(i)
	}
	wg.Wait()
	return allPaths
}

// MonteCarlo 蒙特卡洛模拟.
type MonteCarlo struct {
	gbm *GeometricBrownianMotion
}

func NewMonteCarlo(gbm *GeometricBrownianMotion) *MonteCarlo {
	return &MonteCarlo{gbm: gbm}
}

func (mc *MonteCarlo) CalculateOptionPrice(optionType string, strikePrice decimal.Decimal, steps, paths int, riskFreeRate decimal.Decimal) (decimal.Decimal, error) {
	allPaths := mc.gbm.SimulateMultiplePaths(steps, paths)
	totalPayoff := decimal.Zero
	for _, path := range allPaths {
		finalPrice := path[len(path)-1]
		var payoff decimal.Decimal
		if optionType == "CALL" {
			payoff = finalPrice.Sub(strikePrice)
		} else {
			payoff = strikePrice.Sub(finalPrice)
		}
		if payoff.LessThan(decimal.Zero) {
			payoff = decimal.Zero
		}
		totalPayoff = totalPayoff.Add(payoff)
	}
	avgPayoff := totalPayoff.Div(decimal.NewFromInt(int64(paths)))
	discountFactor := decimal.NewFromFloat(math.Exp(-riskFreeRate.InexactFloat64() * mc.gbm.timeStep.InexactFloat64() * float64(steps)))
	return avgPayoff.Mul(discountFactor), nil
}

// HestonModel 赫斯顿模型 (随机波动率).
type HestonModel struct {
	InitialPrice decimal.Decimal
	InitialVol   decimal.Decimal
	Kappa        decimal.Decimal
	Theta        decimal.Decimal
	Sigma        decimal.Decimal
	Rho          decimal.Decimal
}

func (h *HestonModel) Simulate(steps int, dt decimal.Decimal) []decimal.Decimal {
	prices := make([]decimal.Decimal, steps+1)
	vols := make([]float64, steps+1)
	prices[0] = h.InitialPrice
	vols[0] = h.InitialVol.InexactFloat64()

	dtF := dt.InexactFloat64()
	kappaF := h.Kappa.InexactFloat64()
	thetaF := h.Theta.InexactFloat64()
	sigmaF := h.Sigma.InexactFloat64()
	rhoF := h.Rho.InexactFloat64()

	for i := 1; i <= steps; i++ {
		z1 := cryptoNormFloat64()
		z2 := cryptoNormFloat64()
		zv := rhoF*z1 + math.Sqrt(1-rhoF*rhoF)*z2

		vPrev := math.Max(0, vols[i-1])
		vNext := vPrev + kappaF*(thetaF-vPrev)*dtF + sigmaF*math.Sqrt(vPrev*dtF)*zv
		vols[i] = math.Max(0, vNext)

		pPrev := prices[i-1].InexactFloat64()
		exponent := (0-0.5*vPrev)*dtF + math.Sqrt(vPrev*dtF)*z1
		prices[i] = decimal.NewFromFloat(pPrev * math.Exp(exponent))
	}
	return prices
}
