package domain

import "time"

// Price represents the system-wide price for an asset
type Price struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Symbol    string    `json:"symbol"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	Mid       float64   `json:"mid"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
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
