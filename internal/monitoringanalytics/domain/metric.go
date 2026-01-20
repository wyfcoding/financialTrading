package domain

import (
	"time"

	"gorm.io/gorm"
)

// TradeMetric represents aggregated trade data for a time window
type TradeMetric struct {
	gorm.Model
	Symbol       string    `gorm:"column:symbol;type:varchar(20);index"`
	MetricType   string    `gorm:"column:metric_type;type:varchar(20)"` // e.g., "1min", "1h"
	Timestamp    time.Time `gorm:"column:timestamp;index"`
	TotalVolume  float64   `gorm:"column:total_volume;type:decimal(20,8)"`
	TradeCount   int       `gorm:"column:trade_count;type:int"`
	AveragePrice float64   `gorm:"column:average_price;type:decimal(20,8)"`
}

// TableName overrides
func (TradeMetric) TableName() string {
	return "trade_metrics"
}
