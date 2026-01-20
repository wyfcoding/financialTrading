package domain

import (
	"time"

	"gorm.io/gorm"
)

// Price represents the system-wide price for an asset
// Note: We use gorm.Model for convenience, but for pricing, a simple key-value store or latest-row-only might be better.
// Here we store history, but query mostly latest.
type Price struct {
	gorm.Model
	Symbol    string    `gorm:"column:symbol;type:varchar(20);index;not null"`
	Bid       float64   `gorm:"column:bid;type:decimal(20,8)"`
	Ask       float64   `gorm:"column:ask;type:decimal(20,8)"`
	Mid       float64   `gorm:"column:mid;type:decimal(20,8)"`
	Source    string    `gorm:"column:source;type:varchar(50)"`
	Timestamp time.Time `gorm:"column:timestamp;index"`
}

func (Price) TableName() string {
	return "prices"
}

func NewPrice(symbol string, bid, ask float64, source string) *Price {
	mid := (bid + ask) / 2
	return &Price{
		Symbol:    symbol,
		Bid:       bid,
		Ask:       ask,
		Mid:       mid,
		Source:    source,
		Timestamp: time.Now(),
	}
}
