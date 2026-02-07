package domain

import "time"

// TradeMetric represents aggregated trade data for a time window
type TradeMetric struct {
	ID           uint      `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Symbol       string    `json:"symbol"`
	MetricType   string    `json:"metric_type"` // e.g., "1min", "1h"
	Timestamp    time.Time `json:"timestamp"`
	TotalVolume  float64   `json:"total_volume"`
	TradeCount   int       `json:"trade_count"`
	AveragePrice float64   `json:"average_price"`
}

// TradeMetric is a read model for trade statistics.
