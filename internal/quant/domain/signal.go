package domain

import (
	"time"

	"gorm.io/gorm"
)

type IndicatorType string

const (
	SMA IndicatorType = "SMA"
	RSI IndicatorType = "RSI"
	EMA IndicatorType = "EMA"
)

type Signal struct {
	gorm.Model
	Symbol    string        `gorm:"column:symbol;type:varchar(20);index;not null"`
	Indicator IndicatorType `gorm:"column:indicator;type:varchar(10);not null"`
	Period    int           `gorm:"column:period;not null"`
	Value     float64       `gorm:"column:value;type:decimal(20,8)"`
	Timestamp time.Time     `gorm:"column:timestamp;index"`
}

func (Signal) TableName() string {
	return "signals"
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

func (l *IndicatorLogic) CalculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 0
	}
	// Simplified RSI
	gains := 0.0
	losses := 0.0

	// Consider last N changes
	subset := prices[len(prices)-period-1:]
	for i := 1; i < len(subset); i++ {
		change := subset[i] - subset[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}
