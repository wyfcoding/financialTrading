package application

import "time"

type SubmitOrderCommand struct {
	OrderID  string  `json:"order_id"`
	Symbol   string  `json:"symbol"`
	Side     string  `json:"side"` // "buy" or "sell"
	Type     string  `json:"type"` // "limit" or "market"
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

type OrderBookDTO struct {
	Symbol    string     `json:"symbol"`
	Bids      []LevelDTO `json:"bids"`
	Asks      []LevelDTO `json:"asks"`
	Timestamp time.Time  `json:"timestamp"`
}

type LevelDTO struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

type TradeDTO struct {
	MakerOrderID string    `json:"maker_order_id"`
	TakerOrderID string    `json:"taker_order_id"`
	Symbol       string    `json:"symbol"`
	Price        float64   `json:"price"`
	Quantity     float64   `json:"quantity"`
	Timestamp    time.Time `json:"timestamp"`
}
