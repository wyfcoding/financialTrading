package domain

import (
	"time"

	"github.com/wyfcoding/pkg/eventsourcing"
)

// OrderCreatedEvent 订单创建事件
type OrderCreatedEvent struct {
	eventsourcing.BaseEvent
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

func (e *OrderCreatedEvent) EventType() string     { return "OrderCreated" }
func (e *OrderCreatedEvent) AggregateID() string   { return e.OrderID }
func (e *OrderCreatedEvent) Version() int64        { return e.Ver }
func (e *OrderCreatedEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderCreatedEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderValidatedEvent 订单验证通过事件
type OrderValidatedEvent struct {
	eventsourcing.BaseEvent
	OrderID     string
	UserID      string
	Symbol      string
	ValidatedAt int64
	OccurredOn  time.Time
}

func (e *OrderValidatedEvent) EventType() string     { return "OrderValidated" }
func (e *OrderValidatedEvent) AggregateID() string   { return e.OrderID }
func (e *OrderValidatedEvent) Version() int64        { return e.Ver }
func (e *OrderValidatedEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderValidatedEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderRejectedEvent 订单被拒绝事件
type OrderRejectedEvent struct {
	eventsourcing.BaseEvent
	OrderID    string
	UserID     string
	Symbol     string
	Reason     string
	RejectedAt int64
	OccurredOn time.Time
}

func (e *OrderRejectedEvent) EventType() string     { return "OrderRejected" }
func (e *OrderRejectedEvent) AggregateID() string   { return e.OrderID }
func (e *OrderRejectedEvent) Version() int64        { return e.Ver }
func (e *OrderRejectedEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderRejectedEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderPartiallyFilledEvent 订单部分成交事件
type OrderPartiallyFilledEvent struct {
	eventsourcing.BaseEvent
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

func (e *OrderPartiallyFilledEvent) EventType() string     { return "OrderPartiallyFilled" }
func (e *OrderPartiallyFilledEvent) AggregateID() string   { return e.OrderID }
func (e *OrderPartiallyFilledEvent) Version() int64        { return e.Ver }
func (e *OrderPartiallyFilledEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderPartiallyFilledEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderFilledEvent 订单完全成交事件
type OrderFilledEvent struct {
	eventsourcing.BaseEvent
	OrderID       string
	UserID        string
	Symbol        string
	TotalQuantity float64
	AveragePrice  float64
	FilledAt      int64
	OccurredOn    time.Time
}

func (e *OrderFilledEvent) EventType() string     { return "OrderFilled" }
func (e *OrderFilledEvent) AggregateID() string   { return e.OrderID }
func (e *OrderFilledEvent) Version() int64        { return e.Ver }
func (e *OrderFilledEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderFilledEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderCancelledEvent 订单被取消事件
type OrderCancelledEvent struct {
	eventsourcing.BaseEvent
	OrderID     string
	UserID      string
	Symbol      string
	Reason      string
	CancelledAt int64
	OccurredOn  time.Time
}

func (e *OrderCancelledEvent) EventType() string     { return "OrderCancelled" }
func (e *OrderCancelledEvent) AggregateID() string   { return e.OrderID }
func (e *OrderCancelledEvent) Version() int64        { return e.Ver }
func (e *OrderCancelledEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderCancelledEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderExpiredEvent 订单过期事件
type OrderExpiredEvent struct {
	eventsourcing.BaseEvent
	OrderID    string
	UserID     string
	Symbol     string
	ExpiredAt  int64
	OccurredOn time.Time
}

func (e *OrderExpiredEvent) EventType() string     { return "OrderExpired" }
func (e *OrderExpiredEvent) AggregateID() string   { return e.OrderID }
func (e *OrderExpiredEvent) Version() int64        { return e.Ver }
func (e *OrderExpiredEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderExpiredEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderStatusChangedEvent 订单状态变更事件
type OrderStatusChangedEvent struct {
	eventsourcing.BaseEvent
	OrderID    string
	OldStatus  OrderStatus
	NewStatus  OrderStatus
	UpdatedAt  int64
	OccurredOn time.Time
}

func (e *OrderStatusChangedEvent) EventType() string     { return "OrderStatusChanged" }
func (e *OrderStatusChangedEvent) AggregateID() string   { return e.OrderID }
func (e *OrderStatusChangedEvent) Version() int64        { return e.Ver }
func (e *OrderStatusChangedEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderStatusChangedEvent) OccurredAt() time.Time { return e.OccurredOn }
