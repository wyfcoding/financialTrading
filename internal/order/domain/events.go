package domain

import (
	"time"

	"github.com/wyfcoding/pkg/eventsourcing"
)

const (
	OrderCreatedEventType         = "OrderCreated"
	OrderValidatedEventType       = "OrderValidated"
	OrderRejectedEventType        = "OrderRejected"
	OrderPartiallyFilledEventType = "OrderPartiallyFilled"
	OrderFilledEventType          = "OrderFilled"
	OrderCancelledEventType       = "OrderCancelled"
	OrderExpiredEventType         = "OrderExpired"
	OrderStatusChangedEventType   = "OrderStatusChanged"
)

// OrderCreatedEvent 订单创建事件
type OrderCreatedEvent struct {
	eventsourcing.BaseEvent
	OrderID       string      `json:"order_id"`
	UserID        string      `json:"user_id"`
	Symbol        string      `json:"symbol"`
	Side          OrderSide   `json:"side"`
	Type          OrderType   `json:"type"`
	Price         float64     `json:"price"`
	StopPrice     float64     `json:"stop_price"`
	Quantity      float64     `json:"quantity"`
	TimeInForce   TimeInForce `json:"time_in_force"`
	ParentOrderID string      `json:"parent_order_id"`
	IsOCO         bool        `json:"is_oco"`
	OccurredOn    time.Time   `json:"occurred_on"`
}

func (e *OrderCreatedEvent) EventType() string     { return OrderCreatedEventType }
func (e *OrderCreatedEvent) AggregateID() string   { return e.OrderID }
func (e *OrderCreatedEvent) Version() int64        { return e.Ver }
func (e *OrderCreatedEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderCreatedEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderValidatedEvent 订单验证通过事件
type OrderValidatedEvent struct {
	eventsourcing.BaseEvent
	OrderID     string    `json:"order_id"`
	UserID      string    `json:"user_id"`
	Symbol      string    `json:"symbol"`
	ValidatedAt int64     `json:"validated_at"`
	OccurredOn  time.Time `json:"occurred_on"`
}

func (e *OrderValidatedEvent) EventType() string     { return OrderValidatedEventType }
func (e *OrderValidatedEvent) AggregateID() string   { return e.OrderID }
func (e *OrderValidatedEvent) Version() int64        { return e.Ver }
func (e *OrderValidatedEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderValidatedEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderRejectedEvent 订单被拒绝事件
type OrderRejectedEvent struct {
	eventsourcing.BaseEvent
	OrderID    string    `json:"order_id"`
	UserID     string    `json:"user_id"`
	Symbol     string    `json:"symbol"`
	Reason     string    `json:"reason"`
	RejectedAt int64     `json:"rejected_at"`
	OccurredOn time.Time `json:"occurred_on"`
}

func (e *OrderRejectedEvent) EventType() string     { return OrderRejectedEventType }
func (e *OrderRejectedEvent) AggregateID() string   { return e.OrderID }
func (e *OrderRejectedEvent) Version() int64        { return e.Ver }
func (e *OrderRejectedEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderRejectedEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderPartiallyFilledEvent 订单部分成交事件
type OrderPartiallyFilledEvent struct {
	eventsourcing.BaseEvent
	OrderID           string    `json:"order_id"`
	UserID            string    `json:"user_id"`
	Symbol            string    `json:"symbol"`
	FilledQuantity    float64   `json:"filled_quantity"`
	RemainingQuantity float64   `json:"remaining_quantity"`
	TradePrice        float64   `json:"trade_price"`
	AveragePrice      float64   `json:"average_price"`
	FilledAt          int64     `json:"filled_at"`
	OccurredOn        time.Time `json:"occurred_on"`
}

func (e *OrderPartiallyFilledEvent) EventType() string     { return OrderPartiallyFilledEventType }
func (e *OrderPartiallyFilledEvent) AggregateID() string   { return e.OrderID }
func (e *OrderPartiallyFilledEvent) Version() int64        { return e.Ver }
func (e *OrderPartiallyFilledEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderPartiallyFilledEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderFilledEvent 订单完全成交事件
type OrderFilledEvent struct {
	eventsourcing.BaseEvent
	OrderID       string    `json:"order_id"`
	UserID        string    `json:"user_id"`
	Symbol        string    `json:"symbol"`
	TotalQuantity float64   `json:"total_quantity"`
	AveragePrice  float64   `json:"average_price"`
	FilledAt      int64     `json:"filled_at"`
	OccurredOn    time.Time `json:"occurred_on"`
}

func (e *OrderFilledEvent) EventType() string     { return OrderFilledEventType }
func (e *OrderFilledEvent) AggregateID() string   { return e.OrderID }
func (e *OrderFilledEvent) Version() int64        { return e.Ver }
func (e *OrderFilledEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderFilledEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderCancelledEvent 订单被取消事件
type OrderCancelledEvent struct {
	eventsourcing.BaseEvent
	OrderID     string    `json:"order_id"`
	UserID      string    `json:"user_id"`
	Symbol      string    `json:"symbol"`
	Reason      string    `json:"reason"`
	CancelledAt int64     `json:"cancelled_at"`
	OccurredOn  time.Time `json:"occurred_on"`
}

func (e *OrderCancelledEvent) EventType() string     { return OrderCancelledEventType }
func (e *OrderCancelledEvent) AggregateID() string   { return e.OrderID }
func (e *OrderCancelledEvent) Version() int64        { return e.Ver }
func (e *OrderCancelledEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderCancelledEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderExpiredEvent 订单过期事件
type OrderExpiredEvent struct {
	eventsourcing.BaseEvent
	OrderID    string    `json:"order_id"`
	UserID     string    `json:"user_id"`
	Symbol     string    `json:"symbol"`
	ExpiredAt  int64     `json:"expired_at"`
	OccurredOn time.Time `json:"occurred_on"`
}

func (e *OrderExpiredEvent) EventType() string     { return OrderExpiredEventType }
func (e *OrderExpiredEvent) AggregateID() string   { return e.OrderID }
func (e *OrderExpiredEvent) Version() int64        { return e.Ver }
func (e *OrderExpiredEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderExpiredEvent) OccurredAt() time.Time { return e.OccurredOn }

// OrderStatusChangedEvent 订单状态变更事件
type OrderStatusChangedEvent struct {
	eventsourcing.BaseEvent
	OrderID    string      `json:"order_id"`
	OldStatus  OrderStatus `json:"old_status"`
	NewStatus  OrderStatus `json:"new_status"`
	UpdatedAt  int64       `json:"updated_at"`
	OccurredOn time.Time   `json:"occurred_on"`
}

func (e *OrderStatusChangedEvent) EventType() string     { return OrderStatusChangedEventType }
func (e *OrderStatusChangedEvent) AggregateID() string   { return e.OrderID }
func (e *OrderStatusChangedEvent) Version() int64        { return e.Ver }
func (e *OrderStatusChangedEvent) SetVersion(v int64)    { e.Ver = v }
func (e *OrderStatusChangedEvent) OccurredAt() time.Time { return e.OccurredOn }
