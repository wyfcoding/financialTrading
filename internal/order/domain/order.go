package domain

import (
	"errors"
	"time"

	"github.com/wyfcoding/pkg/eventsourcing"
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
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	eventsourcing.AggregateRoot
	OrderID        string      `json:"order_id"` // UUID
	UserID         string      `json:"user_id"`
	Symbol         string      `json:"symbol"`
	Side           OrderSide   `json:"side"`
	Type           OrderType   `json:"type"`
	Price          float64     `json:"price"`
	StopPrice      float64     `json:"stop_price"`
	Quantity       float64     `json:"quantity"`
	FilledQuantity float64     `json:"filled_quantity"`
	AveragePrice   float64     `json:"average_price"`
	Status         OrderStatus `json:"status"`
	TimeInForce    TimeInForce `json:"time_in_force"`

	// Complex Order Support
	ParentOrderID string `json:"parent_order_id"` // For Bracket/OCO
	IsOCO         bool   `json:"is_oco"`
}

func NewOrder(id, userID, symbol string, side OrderSide, typ OrderType, price, qty float64, stopPrice float64, tif TimeInForce, parentID string, isOCO bool) *Order {
	o := &Order{
		OrderID:       id,
		UserID:        userID,
		Symbol:        symbol,
		Side:          side,
		Type:          typ,
		Price:         price,
		StopPrice:     stopPrice,
		Quantity:      qty,
		Status:        StatusPending,
		TimeInForce:   tif,
		ParentOrderID: parentID,
		IsOCO:         isOCO,
	}
	o.SetID(id)

	o.ApplyChange(&OrderCreatedEvent{
		OrderID:       id,
		UserID:        userID,
		Symbol:        symbol,
		Side:          side,
		Type:          typ,
		Price:         price,
		StopPrice:     stopPrice,
		Quantity:      qty,
		TimeInForce:   tif,
		ParentOrderID: parentID,
		IsOCO:         isOCO,
		OccurredOn:    time.Now(),
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
		o.StopPrice = e.StopPrice
		o.Quantity = e.Quantity
		o.TimeInForce = e.TimeInForce
		o.ParentOrderID = e.ParentOrderID
		o.IsOCO = e.IsOCO
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
