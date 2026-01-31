package domain

import (
	"time"
)

// OrderCreatedEvent 订单创建事件
type OrderCreatedEvent struct {
	OrderID       string
	UserID        string
	Symbol        string
	Side          OrderSide
	Type          OrderType
	Price         float64
	StopPrice     float64
	Quantity      float64
	TimeInForce   TimeInForce
	ParentOrderID string
	IsOCO         bool
	OccurredOn    time.Time
}

// OrderValidatedEvent 订单验证通过事件
type OrderValidatedEvent struct {
	OrderID     string
	UserID      string
	Symbol      string
	ValidatedAt int64
	OccurredOn  time.Time
}

// OrderRejectedEvent 订单被拒绝事件
type OrderRejectedEvent struct {
	OrderID    string
	UserID     string
	Symbol     string
	Reason     string
	RejectedAt int64
	OccurredOn time.Time
}

// OrderPartiallyFilledEvent 订单部分成交事件
type OrderPartiallyFilledEvent struct {
	OrderID           string
	UserID            string
	Symbol            string
	FilledQuantity    float64
	RemainingQuantity float64
	TradePrice        float64
	AveragePrice      float64
	FilledAt          int64
	OccurredOn        time.Time
}

// OrderFilledEvent 订单完全成交事件
type OrderFilledEvent struct {
	OrderID       string
	UserID        string
	Symbol        string
	TotalQuantity float64
	AveragePrice  float64
	FilledAt      int64
	OccurredOn    time.Time
}

// OrderCancelledEvent 订单被取消事件
type OrderCancelledEvent struct {
	OrderID     string
	UserID      string
	Symbol      string
	Reason      string
	CancelledAt int64
	OccurredOn  time.Time
}

// OrderExpiredEvent 订单过期事件
type OrderExpiredEvent struct {
	OrderID    string
	UserID     string
	Symbol     string
	ExpiredAt  int64
	OccurredOn time.Time
}

// OrderStatusChangedEvent 订单状态变更事件
type OrderStatusChangedEvent struct {
	OrderID    string
	OldStatus  OrderStatus
	NewStatus  OrderStatus
	UpdatedAt  int64
	OccurredOn time.Time
}
