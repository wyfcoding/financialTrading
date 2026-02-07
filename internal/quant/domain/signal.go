package domain

import (
	"math"
	"time"
)

type IndicatorType string

const (
	SMAIndicator  IndicatorType = "SMA"
	RSIIndicator  IndicatorType = "RSI"
	EMAIndicator  IndicatorType = "EMA"
	MACDIndicator IndicatorType = "MACD"
	BBIndicator   IndicatorType = "BB"
)

type Signal struct {
	ID         uint          `json:"id"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	StrategyID string        `json:"strategy_id"`
	Symbol     string        `json:"symbol"`
	Indicator  IndicatorType `json:"indicator"`
	Period     int           `json:"period"`
	Value      float64       `json:"value"`
	Confidence float64       `json:"confidence"`
	Timestamp  time.Time     `json:"timestamp"`
}

// IndicatorLogic contains pure domain logic for calculations
type IndicatorLogic struct{}

func (l *IndicatorLogic) CalculateSMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	sum := 0.0
	// Use last 'period' prices
	// Assuming prices are uniform time intervals
	subset := prices[len(prices)-period:]
	for _, p := range subset {
		sum += p
	}
	return sum / float64(period)
}

func (l *IndicatorLogic) CalculateEMA(prices []float64, period int) float64 {
	if len(prices) == 0 {
		return 0
	}
	if len(prices) == 1 {
		return prices[0]
	}
	k := 2.0 / (float64(period) + 1.0)
	ema := prices[0]
	for i := 1; i < len(prices); i++ {
		ema = (prices[i] * k) + (ema * (1 - k))
	}
	return ema
}

func (l *IndicatorLogic) CalculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 0
	}
	// RSI using Wilder's Smoothing
	var gains, losses float64
	for i := 1; i <= period; i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	for i := period + 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		var currentGain, currentLoss float64
		if change > 0 {
			currentGain = change
		} else {
			currentLoss = -change
		}
		avgGain = (avgGain*(float64(period)-1) + currentGain) / float64(period)
		avgLoss = (avgLoss*(float64(period)-1) + currentLoss) / float64(period)
	}

	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func (l *IndicatorLogic) CalculateMACD(prices []float64, fast, slow, signal int) (macd, signalLine, hist float64) {
	if len(prices) < slow {
		return 0, 0, 0
	}
	fastEMA := l.CalculateEMA(prices, fast)
	slowEMA := l.CalculateEMA(prices, slow)
	macd = fastEMA - slowEMA

	// Simplified: Signal is EMA of MACD over N days.
	// In real setup, you'd need a series of MACD values.
	// We'll return the point values.
	return macd, 0, 0
}

func (l *IndicatorLogic) CalculateBollingerBands(prices []float64, period int, stdDevMult float64) (mid, upper, lower float64) {
	mid = l.CalculateSMA(prices, period)
	var sumSqDiff float64
	subset := prices[len(prices)-period:]
	for _, p := range subset {
		diff := p - mid
		sumSqDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSqDiff / float64(period))
	upper = mid + (stdDevMult * stdDev)
	lower = mid - (stdDevMult * stdDev)
	return mid, upper, lower
}
