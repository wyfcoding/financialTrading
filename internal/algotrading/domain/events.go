// Package domain 算法交易服务领域事件
package domain

import "time"

type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

// StrategyStartedEvent 策略启动事件
type StrategyStartedEvent struct {
	StrategyID string    `json:"strategy_id"`
	UserID     uint64    `json:"user_id"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e *StrategyStartedEvent) EventName() string     { return "algotrading.strategy_started" }
func (e *StrategyStartedEvent) OccurredAt() time.Time { return e.Timestamp }

// StrategyStoppedEvent 策略停止事件
type StrategyStoppedEvent struct {
	StrategyID string    `json:"strategy_id"`
	UserID     uint64    `json:"user_id"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e *StrategyStoppedEvent) EventName() string     { return "algotrading.strategy_stopped" }
func (e *StrategyStoppedEvent) OccurredAt() time.Time { return e.Timestamp }
