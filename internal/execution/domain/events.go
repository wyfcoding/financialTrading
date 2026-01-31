package domain

import (
	"time"

	"github.com/wyfcoding/pkg/eventsourcing"
)

// TradeExecutedEvent 成交事件
type TradeExecutedEvent struct {
	eventsourcing.BaseEvent
	TradeID  string
	OrderID  string
	UserID   string
	Symbol   string
	Quantity string
	Price    string
	Time     int64
}

func (e *TradeExecutedEvent) EventType() string     { return "TradeExecuted" }
func (e *TradeExecutedEvent) AggregateID() string   { return e.TradeID }
func (e *TradeExecutedEvent) Version() int64        { return e.Ver }
func (e *TradeExecutedEvent) SetVersion(v int64)    { e.Ver = v }
func (e *TradeExecutedEvent) OccurredAt() time.Time { return time.Unix(e.Time, 0) }

// AlgoOrderStartedEvent 算法订单开始
type AlgoOrderStartedEvent struct {
	eventsourcing.BaseEvent
	AlgoID    string
	UserID    string
	Symbol    string
	AlgoType  string
	TotalQty  string
	StartTime int64
}

func (e *AlgoOrderStartedEvent) EventType() string     { return "AlgoOrderStarted" }
func (e *AlgoOrderStartedEvent) AggregateID() string   { return e.AlgoID }
func (e *AlgoOrderStartedEvent) Version() int64        { return e.Ver }
func (e *AlgoOrderStartedEvent) SetVersion(v int64)    { e.Ver = v }
func (e *AlgoOrderStartedEvent) OccurredAt() time.Time { return time.Unix(e.StartTime, 0) }
