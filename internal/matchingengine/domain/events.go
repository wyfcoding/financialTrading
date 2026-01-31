package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// MatchingEvent 撮合引擎领域事件接口
type MatchingEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// BaseEvent 基础事件结构
type BaseEvent struct {
	Timestamp time.Time
}

// OccurredAt 返回事件发生时间
func (e BaseEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// OrderReceivedEvent 订单接收事件
type OrderReceivedEvent struct {
	BaseEvent
	OrderID   string
	UserID    string
	Symbol    string
	Side      string
	Quantity  decimal.Decimal
	Price     decimal.Decimal
	OrderType string
}

// EventType 返回事件类型
func (e OrderReceivedEvent) EventType() string { return "OrderReceived" }

// OrderMatchedEvent 订单成交事件
type OrderMatchedEvent struct {
	BaseEvent
	MatchID   string
	BuyOrder  string
	SellOrder string
	Symbol    string
	Quantity  decimal.Decimal
	Price     decimal.Decimal
	Timestamp time.Time
}

// EventType 返回事件类型
func (e OrderMatchedEvent) EventType() string { return "OrderMatched" }

// OrderPartiallyFilledEvent 订单部分成交事件
type OrderPartiallyFilledEvent struct {
	BaseEvent
	OrderID     string
	Symbol      string
	FilledQty   decimal.Decimal
	RemainingQty decimal.Decimal
	Price       decimal.Decimal
}

// EventType 返回事件类型
func (e OrderPartiallyFilledEvent) EventType() string { return "OrderPartiallyFilled" }

// OrderFilledEvent 订单完全成交事件
type OrderFilledEvent struct {
	BaseEvent
	OrderID   string
	Symbol    string
	Quantity  decimal.Decimal
	Price     decimal.Decimal
}

// EventType 返回事件类型
func (e OrderFilledEvent) EventType() string { return "OrderFilled" }

// OrderCanceledEvent 订单取消事件
type OrderCanceledEvent struct {
	BaseEvent
	OrderID   string
	Symbol    string
	Reason    string
}

// EventType 返回事件类型
func (e OrderCanceledEvent) EventType() string { return "OrderCanceled" }

// OrderRejectedEvent 订单拒绝事件
type OrderRejectedEvent struct {
	BaseEvent
	OrderID   string
	Symbol    string
	Reason    string
}

// EventType 返回事件类型
func (e OrderRejectedEvent) EventType() string { return "OrderRejected" }

// AuctionStartedEvent 集合竞价开始事件
type AuctionStartedEvent struct {
	BaseEvent
	AuctionID string
	Symbol    string
	StartTime time.Time
}

// EventType 返回事件类型
func (e AuctionStartedEvent) EventType() string { return "AuctionStarted" }

// AuctionEndedEvent 集合竞价结束事件
type AuctionEndedEvent struct {
	BaseEvent
	AuctionID string
	Symbol    string
	EndTime   time.Time
	Price     decimal.Decimal
	Quantity  decimal.Decimal
}

// EventType 返回事件类型
func (e AuctionEndedEvent) EventType() string { return "AuctionEnded" }

// OrderBookUpdatedEvent 订单簿更新事件
type OrderBookUpdatedEvent struct {
	BaseEvent
	Symbol    string
	BestBid   decimal.Decimal
	BestAsk   decimal.Decimal
	BidSize   decimal.Decimal
	AskSize   decimal.Decimal
}

// EventType 返回事件类型
func (e OrderBookUpdatedEvent) EventType() string { return "OrderBookUpdated" }
