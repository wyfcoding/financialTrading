package domain

import "time"

const (
	SettlementCreatedEventType   = "clearing.settlement.created"
	SettlementCompletedEventType = "clearing.settlement.completed"
	SettlementFailedEventType    = "clearing.settlement.failed"
	FXHedgeExecutedEventType     = "clearing.fx_hedge.executed"
)

// ClearingEvent 清算领域事件接口
type ClearingEvent interface {
	EventType() string
	OccurredAt() time.Time
}

type BaseEvent struct {
	Timestamp time.Time `json:"timestamp"`
}

func (e BaseEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// SettlementCreatedEvent 结算单创建事件
type SettlementCreatedEvent struct {
	BaseEvent
	SettlementID string `json:"settlement_id"`
	TradeID      string `json:"trade_id"`
	TotalAmount  string `json:"total_amount"`
}

func (e SettlementCreatedEvent) EventType() string { return "SettlementCreated" }

// SettlementCompletedEvent 结算完成事件
type SettlementCompletedEvent struct {
	BaseEvent
	SettlementID string `json:"settlement_id"`
	TradeID      string `json:"trade_id"`
}

func (e SettlementCompletedEvent) EventType() string { return "SettlementCompleted" }

// SettlementFailedEvent 结算失败事件
type SettlementFailedEvent struct {
	BaseEvent
	SettlementID string `json:"settlement_id"`
	TradeID      string `json:"trade_id"`
	Reason       string `json:"reason"`
}

func (e SettlementFailedEvent) EventType() string { return "SettlementFailed" }
