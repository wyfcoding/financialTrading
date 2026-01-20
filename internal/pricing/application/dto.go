package application

import "time"

type PriceDTO struct {
	Symbol    string    `json:"symbol"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	Mid       float64   `json:"mid"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
}
