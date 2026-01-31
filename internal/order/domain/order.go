package domain

import (
	"errors"
	"time"

	"github.com/wyfcoding/pkg/eventsourcing"
	"gorm.io/gorm"
)

type OrderType string
type OrderSide string
type OrderStatus string

const (
	TypeLimit      OrderType = "limit"
	TypeMarket     OrderType = "market"
	TypeStopLimit  OrderType = "stop_limit"
	TypeStopMarket OrderType = "stop_market"
	TypeTrailing   OrderType = "trailing_stop"
)

type TimeInForce string

const (
	GTC TimeInForce = "GTC" // Good 'Til Cancelled
	IOC TimeInForce = "IOC" // Immediate Or Cancel
	FOK TimeInForce = "FOK" // Fill Or Kill
)

const (
	SideBuy  OrderSide = "buy"
	SideSell OrderSide = "sell"

	StatusPending         OrderStatus = "pending"
	StatusValidated       OrderStatus = "validated"
	StatusRejected        OrderStatus = "rejected"
	StatusPartiallyFilled OrderStatus = "partially_filled"
	StatusFilled          OrderStatus = "filled"
	StatusCancelled       OrderStatus = "cancelled"
	StatusExpired         OrderStatus = "expired"
)

// Order represents an OMS order
type Order struct {
	gorm.Model
	eventsourcing.AggregateRoot
	OrderID        string      `gorm:"column:order_id;type:varchar(36);uniqueIndex;not null" json:"order_id"` // UUID
	UserID         string      `gorm:"column:user_id;type:varchar(50);index;not null" json:"user_id"`
	Symbol         string      `gorm:"column:symbol;type:varchar(20);not null" json:"symbol"`
	Side           OrderSide   `gorm:"column:side;type:varchar(10);not null" json:"side"`
	Type           OrderType   `gorm:"column:type;type:varchar(20);not null" json:"type"`
	Price          float64     `gorm:"column:price;type:decimal(20,8)" json:"price"`
	StopPrice      float64     `gorm:"column:stop_price;type:decimal(20,8)" json:"stop_price"`
	Quantity       float64     `gorm:"column:quantity;type:decimal(20,8);not null" json:"quantity"`
	FilledQuantity float64     `gorm:"column:filled_quantity;type:decimal(20,8);default:0" json:"filled_quantity"`
	AveragePrice   float64     `gorm:"column:average_price;type:decimal(20,8);default:0" json:"average_price"`
	Status         OrderStatus `gorm:"column:status;type:varchar(20);index;not null;default:'pending'" json:"status"`
	TimeInForce    TimeInForce `gorm:"column:tif;type:varchar(10);default:'GTC'" json:"tif"`

	// Complex Order Support
	ParentOrderID string `gorm:"column:parent_id;type:varchar(36);index" json:"parent_id"` // For Bracket/OCO
	IsOCO         bool   `gorm:"column:is_oco" json:"is_oco"`
}

func (Order) TableName() string {
	return "orders"
}

func NewOrder(id, userID, symbol string, side OrderSide, typ OrderType, price, qty float64) *Order {
	o := &Order{
		OrderID:  id,
		UserID:   userID,
		Symbol:   symbol,
		Side:     side,
		Type:     typ,
		Price:    price,
		Quantity: qty,
		Status:   StatusPending,
	}
	o.SetID(id)

	o.ApplyChange(&OrderCreatedEvent{
		OrderID:    id,
		UserID:     userID,
		Symbol:     symbol,
		Side:       side,
		Type:       typ,
		Price:      price,
		Quantity:   qty,
		OccurredOn: time.Now(),
	})
	return o
}

// Apply 实现了 eventsourcing.EventApplier 接口
func (o *Order) Apply(event eventsourcing.DomainEvent) {
	switch e := event.(type) {
	case *OrderCreatedEvent:
		o.OrderID = e.OrderID
		o.UserID = e.UserID
		o.Symbol = e.Symbol
		o.Side = e.Side
		o.Type = e.Type
		o.Price = e.Price
		o.Quantity = e.Quantity
		o.Status = StatusPending
	case *OrderValidatedEvent:
		o.Status = StatusValidated
	case *OrderRejectedEvent:
		o.Status = StatusRejected
	case *OrderPartiallyFilledEvent:
		o.FilledQuantity = e.FilledQuantity
		o.AveragePrice = e.AveragePrice
		o.Status = StatusPartiallyFilled
	case *OrderFilledEvent:
		o.FilledQuantity = e.TotalQuantity
		o.AveragePrice = e.AveragePrice
		o.Status = StatusFilled
	case *OrderCancelledEvent:
		o.Status = StatusCancelled
	}
}

// Validate performs basic static validation
func (o *Order) Validate() error {
	if o.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	if o.Type == TypeLimit && o.Price <= 0 {
		return errors.New("price must be positive for limit orders")
	}
	return nil
}

// MarkValidated transitions to Validated state
func (o *Order) MarkValidated() {
	if o.Status == StatusPending {
		o.ApplyChange(&OrderValidatedEvent{
			OrderID:     o.OrderID,
			UserID:      o.UserID,
			Symbol:      o.Symbol,
			ValidatedAt: time.Now().UnixNano(),
			OccurredOn:  time.Now(),
		})
	}
}

// UpdateExecution updates order with execution report
func (o *Order) UpdateExecution(filledQty, tradePrice float64) {
	// Simple average price calculation
	totalValue := (o.AveragePrice * o.FilledQuantity) + (tradePrice * filledQty)
	newFilledQty := o.FilledQuantity + filledQty
	var newAvgPrice float64
	if newFilledQty > 0 {
		newAvgPrice = totalValue / newFilledQty
	}

	if newFilledQty >= o.Quantity {
		o.ApplyChange(&OrderFilledEvent{
			OrderID:       o.OrderID,
			UserID:        o.UserID,
			Symbol:        o.Symbol,
			TotalQuantity: newFilledQty,
			AveragePrice:  newAvgPrice,
			FilledAt:      time.Now().UnixNano(),
			OccurredOn:    time.Now(),
		})
	} else {
		o.ApplyChange(&OrderPartiallyFilledEvent{
			OrderID:           o.OrderID,
			UserID:            o.UserID,
			Symbol:            o.Symbol,
			FilledQuantity:    newFilledQty,
			RemainingQuantity: o.Quantity - newFilledQty,
			TradePrice:        tradePrice,
			AveragePrice:      newAvgPrice,
			FilledAt:          time.Now().UnixNano(),
			OccurredOn:        time.Now(),
		})
	}
}
