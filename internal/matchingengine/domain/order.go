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
// 持久化映射在基础设施层完成

type Order struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	OrderID   string    `json:"order_id"`
	Symbol    string    `json:"symbol"`
	Side      OrderSide `json:"side"`
	Type      OrderType `json:"type"`
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`

	FilledQty float64 `json:"filled_qty"`
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
// 持久化映射在基础设施层完成

type Trade struct {
	ID          uint      `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	TradeID     string    `json:"trade_id"`
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
	Symbol      string    `json:"symbol"`
	Price       float64   `json:"price"`
	Quantity    float64   `json:"quantity"`
	Timestamp   time.Time `json:"timestamp"`
}

var orderPool = sync.Pool{
	New: func() interface{} {
		return &Order{}
	},
}

// NewOrder creates or acquires an order instance from the pool
func NewOrder(orderID, symbol string, side OrderSide, typ OrderType, price, qty float64) *Order {
	o := orderPool.Get().(*Order)
	o.OrderID = orderID
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
	o.OrderID = ""
	o.Symbol = ""
	o.Side = 0
	o.Type = 0
	o.Price = 0
	o.Quantity = 0
	o.FilledQty = 0
	orderPool.Put(o)
}
