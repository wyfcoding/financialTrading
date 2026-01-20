package domain

import (
	"time"
)

// ClearingEvent 清算领域事件接口
type ClearingEvent interface {
	EventType() string
	OccurredAt() time.Time
}

type BaseEvent struct {
	Timestamp time.Time
}

func (e BaseEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// SettlementCreatedEvent 结算单创建事件
type SettlementCreatedEvent struct {
	BaseEvent
	SettlementID string
	TradeID      string
	TotalAmount  string
}

func (e SettlementCreatedEvent) EventType() string { return "SettlementCreated" }

// SettlementCompletedEvent 结算完成事件
type SettlementCompletedEvent struct {
	BaseEvent
	SettlementID string
	TradeID      string
}

func (e SettlementCompletedEvent) EventType() string { return "SettlementCompleted" }

// SettlementFailedEvent 结算失败事件
type SettlementFailedEvent struct {
	BaseEvent
	SettlementID string
	TradeID      string
	Reason       string
}

func (e SettlementFailedEvent) EventType() string { return "SettlementFailed" }
