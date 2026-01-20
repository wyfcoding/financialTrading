package domain

import (
	"sync"
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
	return o.RemainingQty() <= 0
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

var orderPool = sync.Pool{
	New: func() interface{} {
		return &Order{}
	},
}

// NewOrder creates or acquires an order instance from the pool
func NewOrder(id, symbol string, side OrderSide, typ OrderType, price, qty float64) *Order {
	o := orderPool.Get().(*Order)
	o.ID = id
	o.Symbol = symbol
	o.Side = side
	o.Type = typ
	o.Price = price
	o.Quantity = qty
	o.FilledQty = 0
	o.Timestamp = time.Now()
	return o
}

// ReleaseOrder returns an order instance back to the pool
func ReleaseOrder(o *Order) {
	if o == nil {
		return
	}
	o.ID = ""
	o.Symbol = ""
	o.Side = 0
	o.Type = 0
	o.Price = 0
	o.Quantity = 0
	o.FilledQty = 0
	orderPool.Put(o)
}
