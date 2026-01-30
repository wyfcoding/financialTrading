package domain

import (
	"sync"
	"time"

	"gorm.io/gorm"
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
	gorm.Model
	OrderID   string    `gorm:"column:order_id;type:varchar(32);uniqueIndex;not null;comment:订单ID"`
	Symbol    string    `gorm:"column:symbol;type:varchar(20);index;not null;comment:标的"`
	Side      OrderSide `gorm:"column:side;type:tinyint;not null;comment:方向"`
	Type      OrderType `gorm:"column:type;type:tinyint;not null;comment:类型"`
	Price     float64   `gorm:"column:price;type:double;not null;comment:价格"`
	Quantity  float64   `gorm:"column:quantity;type:double;not null;comment:数量"`
	Timestamp time.Time `gorm:"column:timestamp;not null;comment:时间"`

	// Working state
	FilledQty float64 `gorm:"column:filled_qty;type:double;default:0;comment:已成交量"`
}

func (Order) TableName() string {
	return "matching_orders"
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
	gorm.Model
	TradeID     string    `gorm:"column:trade_id;type:varchar(32);uniqueIndex;not null;comment:成交ID"`
	BuyOrderID  string    `gorm:"column:buy_order_id;type:varchar(32);index;not null;comment:买方订单ID"`
	SellOrderID string    `gorm:"column:sell_order_id;type:varchar(32);index;not null;comment:卖方订单ID"`
	Symbol      string    `gorm:"column:symbol;type:varchar(20);index;not null;comment:标的"`
	Price       float64   `gorm:"column:price;type:double;not null;comment:价格"`
	Quantity    float64   `gorm:"column:quantity;type:double;not null;comment:数量"`
	Timestamp   time.Time `gorm:"column:timestamp;not null;comment:时间"`
}

func (Trade) TableName() string {
	return "matching_trades"
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
