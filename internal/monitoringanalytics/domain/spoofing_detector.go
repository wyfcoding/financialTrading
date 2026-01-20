package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algorithm/surveillance"
)

// OrderEvent 监控服务接收到的订单事件简报
type OrderEvent struct {
	UserID    string
	Symbol    string
	OrderID   string
	Price     decimal.Decimal
	Quantity  decimal.Decimal
	Type      string // "PLACE", "CANCEL", "FILL"
	Timestamp time.Time
}

// SpoofingDetector 市场操纵检测器
type SpoofingDetector struct {
	history map[string][]surveillance.MarketEvent
	engine  *surveillance.SurveillanceEngine
}

func NewSpoofingDetector(threshold decimal.Decimal, window time.Duration) *SpoofingDetector {
	return &SpoofingDetector{
		history: make(map[string][]surveillance.MarketEvent),
		engine:  &surveillance.SurveillanceEngine{Threshold: threshold, Window: window},
	}
}

// AbuseScore 违规评分结果
type AbuseScore struct {
	UserID       string
	Score        float64
	Reason       string
	IsSuspicious bool
}

// ProcessEvent 处理订单事件并实时检测操纵风险
func (d *SpoofingDetector) ProcessEvent(event OrderEvent) AbuseScore {
	ev := surveillance.MarketEvent{
		UserID:    event.UserID,
		Type:      event.Type,
		Price:     event.Price,
		Quantity:  event.Quantity,
		Timestamp: event.Timestamp,
	}
	d.history[event.UserID] = append(d.history[event.UserID], ev)

	d.cleanup(event.UserID, event.Timestamp)

	score, reason := d.engine.Analyze(d.history[event.UserID])

	return AbuseScore{
		UserID:       event.UserID,
		Score:        score,
		Reason:       reason,
		IsSuspicious: score > 0.7,
	}
}

func (d *SpoofingDetector) cleanup(userID string, now time.Time) {
	events := d.history[userID]
	var valid []surveillance.MarketEvent
	cutoff := now.Add(-d.engine.Window)

	for _, e := range events {
		if e.Timestamp.After(cutoff) {
			valid = append(valid, e)
		}
	}
	d.history[userID] = valid
}

// End of Spoofing Detector implementation
