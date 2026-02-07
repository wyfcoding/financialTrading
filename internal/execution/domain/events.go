package domain

import (
	"time"

	"github.com/wyfcoding/pkg/eventsourcing"
)

const (
	TradeExecutedEventType    = "execution.trade.executed"
	AlgoOrderStartedEventType = "execution.algo.started"
)

// TradeExecutedEvent 成交事件
type TradeExecutedEvent struct {
	eventsourcing.BaseEvent
	TradeID  string `json:"trade_id"`
	OrderID  string `json:"order_id"`
	UserID   string `json:"user_id"`
	Symbol   string `json:"symbol"`
	Quantity string `json:"quantity"`
	Price    string `json:"price"`
	Time     int64  `json:"time"`
}

func (e *TradeExecutedEvent) EventType() string     { return "TradeExecuted" }
func (e *TradeExecutedEvent) AggregateID() string   { return e.TradeID }
func (e *TradeExecutedEvent) Version() int64        { return e.Ver }
func (e *TradeExecutedEvent) SetVersion(v int64)    { e.Ver = v }
func (e *TradeExecutedEvent) OccurredAt() time.Time { return time.Unix(e.Time, 0) }

// AlgoOrderStartedEvent 算法订单开始
type AlgoOrderStartedEvent struct {
	eventsourcing.BaseEvent
	AlgoID    string `json:"algo_id"`
	UserID    string `json:"user_id"`
	Symbol    string `json:"symbol"`
	AlgoType  string `json:"algo_type"`
	TotalQty  string `json:"total_qty"`
	StartTime int64  `json:"start_time"`
}

func (e *AlgoOrderStartedEvent) EventType() string     { return "AlgoOrderStarted" }
func (e *AlgoOrderStartedEvent) AggregateID() string   { return e.AlgoID }
func (e *AlgoOrderStartedEvent) Version() int64        { return e.Ver }
func (e *AlgoOrderStartedEvent) SetVersion(v int64)    { e.Ver = v }
func (e *AlgoOrderStartedEvent) OccurredAt() time.Time { return time.Unix(e.StartTime, 0) }
