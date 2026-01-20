package domain

import (
	"time"
)

// ExecutionEvent 领域事件接口
type ExecutionEvent interface {
	EventType() string
	OccurredAt() time.Time
}

type BaseEvent struct {
	Timestamp time.Time
}

func (e BaseEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// TradeExecutedEvent 成交事件
type TradeExecutedEvent struct {
	BaseEvent
	TradeID  string
	OrderID  string
	Symbol   string
	Quantity string
	Price    string
}

func (e TradeExecutedEvent) EventType() string { return "TradeExecuted" }

// AlgoOrderStartedEvent 算法订单开始
type AlgoOrderStartedEvent struct {
	BaseEvent
	AlgoID string
	Type   string
}

func (e AlgoOrderStartedEvent) EventType() string { return "AlgoOrderStarted" }
