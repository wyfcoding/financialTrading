package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type OrderBookItem struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

type OrderBook struct {
	Symbol    string
	Bids      []OrderBookItem
	Asks      []OrderBookItem
	Timestamp time.Time
}
