package domain

import (
	"math"
	"math/rand"
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

// NewGBM creates a new GBM generator
func NewGBM(drift, volatility float64, seed int64) *GeometricBrownianMotion {
	return &GeometricBrownianMotion{
		Drift:      drift,
		Volatility: volatility,
		Rand:       rand.New(rand.NewSource(seed)),
	}
}

// Next calculates the next price based on GBM: S(t+dt) = S(t) * exp((mu - 0.5*sigma^2)*dt + sigma*sqrt(dt)*Z)
func (gbm *GeometricBrownianMotion) Next(currentPrice float64, dt float64) float64 {
	z := gbm.Rand.NormFloat64()
	return currentPrice * math.Exp((gbm.Drift-0.5*gbm.Volatility*gbm.Volatility)*dt+gbm.Volatility*math.Sqrt(dt)*z)
}
