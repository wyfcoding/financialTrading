package application

import "time"

type MetricDTO struct {
	Symbol       string    `json:"symbol"`
	Timestamp    time.Time `json:"timestamp"`
	TotalVolume  float64   `json:"total_volume"`
	TradeCount   int       `json:"trade_count"`
	AveragePrice float64   `json:"average_price"`
}

type AlertDTO struct {
	ID        uint      `json:"id"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
