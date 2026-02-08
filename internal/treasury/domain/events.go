// Package domain 资金服务领域事件
package domain

import "time"

type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

// FundsDepositedEvent 资金入金事件
type FundsDepositedEvent struct {
	AccountID uint64    `json:"account_id"`
	Amount    int64     `json:"amount"`
	Balance   int64     `json:"balance"`
	Timestamp time.Time `json:"timestamp"`
}

func (e *FundsDepositedEvent) EventName() string     { return "treasury.funds_deposited" }
func (e *FundsDepositedEvent) OccurredAt() time.Time { return e.Timestamp }

// FundsFrozenEvent 资金冻结事件
type FundsFrozenEvent struct {
	AccountID uint64    `json:"account_id"`
	Amount    int64     `json:"amount"`
	Frozen    int64     `json:"frozen"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

func (e *FundsFrozenEvent) EventName() string     { return "treasury.funds_frozen" }
func (e *FundsFrozenEvent) OccurredAt() time.Time { return e.Timestamp }

// FundsUnfrozenEvent 资金解冻事件
type FundsUnfrozenEvent struct {
	AccountID uint64    `json:"account_id"`
	Amount    int64     `json:"amount"`
	Frozen    int64     `json:"frozen"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

func (e *FundsUnfrozenEvent) EventName() string     { return "treasury.funds_unfrozen" }
func (e *FundsUnfrozenEvent) OccurredAt() time.Time { return e.Timestamp }

// FundsDeductedEvent 资金扣减事件
type FundsDeductedEvent struct {
	AccountID uint64    `json:"account_id"`
	Amount    int64     `json:"amount"`
	Balance   int64     `json:"balance"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

func (e *FundsDeductedEvent) EventName() string     { return "treasury.funds_deducted" }
func (e *FundsDeductedEvent) OccurredAt() time.Time { return e.Timestamp }
