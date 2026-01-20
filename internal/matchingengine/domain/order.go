package domain

import (
	"time"
)

type OrderSide int

const (
	SideBuy OrderSide = iota + 1
	SideSell
)

type OrderType int

const (
	TypeLimit OrderType = iota + 1
	TypeMarket
)

// Order represents an internal order in the matching engine
type Order struct {
	ID        string
	Symbol    string
	Side      OrderSide
	Type      OrderType
	Price     float64
	Quantity  float64
	Timestamp time.Time

	// Working state
	FilledQty float64
}

// RemainingQty returns the remaining quantity to be filled
func (o *Order) RemainingQty() float64 {
	return o.Quantity - o.FilledQty
}

// IsFilled checks if the order is completely filled
func (o *Order) IsFilled() bool {
	return o.RemainingQty() <= 0 // Float epsilon check might be needed in real prod
}

// Trade represents a successful match
type Trade struct {
	MakerOrderID string
	TakerOrderID string
	Symbol       string
	Price        float64
	Quantity     float64
	Timestamp    time.Time
}

// NewOrder creates a new order instance
func NewOrder(id, symbol string, side OrderSide, typ OrderType, price, qty float64) *Order {
	return &Order{
		ID:        id,
		Symbol:    symbol,
		Side:      side,
		Type:      typ,
		Price:     price,
		Quantity:  qty,
		Timestamp: time.Now(),
	}
}
