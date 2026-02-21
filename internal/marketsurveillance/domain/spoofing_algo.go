// 变更说明：
// 从 pkg/algos/surveillance/spoofing.go 迁移。
// 实现了欺诈与层叠 (Spoofing/Layering) 识别算法。
package domain

import (
	"math"
	"time"

	"github.com/shopspring/decimal"
)

type MarketEvent struct {
	Price     decimal.Decimal
	Quantity  decimal.Decimal
	Timestamp time.Time
	UserID    string
	Type      string // PLACE, CANCEL, FILL
}

type Engine struct {
	Threshold decimal.Decimal
	Window    time.Duration
}

func (e *Engine) Analyze(events []MarketEvent) (score float64, reason string) {
	if len(events) == 0 {
		return 0, "No events"
	}

	var cancelCount, fillCount int
	var largeCancels int
	levels := make(map[string]int)

	for _, ev := range events {
		switch ev.Type {
		case "CANCEL":
			cancelCount++
			if ev.Quantity.GreaterThanOrEqual(e.Threshold) {
				largeCancels++
			}
		case "FILL":
			fillCount++
		case "PLACE":
			if ev.Quantity.GreaterThanOrEqual(e.Threshold) {
				levels[ev.Price.String()]++
			}
		}
	}

	spoofScore := 0.0
	if largeCancels > 0 {
		ltr := float64(fillCount) / float64(largeCancels+fillCount)
		if ltr < 0.1 {
			spoofScore = 0.8
		}
	}

	layerScore := 0.0
	if len(levels) >= 3 {
		layerScore = 0.6
	}

	score = math.Max(spoofScore, layerScore)
	if score > 0.7 {
		reason = "Aggressive market manipulation pattern"
	} else {
		reason = "Normal market participation"
	}

	return score, reason
}
