package domain

import (
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/eventsourcing"
)

// AccountCreatedEvent 开户事件
type AccountCreatedEvent struct {
	eventsourcing.BaseEvent
	AccountID   string
	UserID      string
	AccountType string
	Currency    string
}

func (e *AccountCreatedEvent) EventType() string { return "AccountCreated" }

// FundsDepositedEvent 充值事件
type FundsDepositedEvent struct {
	eventsourcing.BaseEvent
	AccountID string
	Amount    decimal.Decimal
	Balance   decimal.Decimal
}

func (e *FundsDepositedEvent) EventType() string { return "FundsDeposited" }

// FundsWithdrawnEvent 提现事件
type FundsWithdrawnEvent struct {
	eventsourcing.BaseEvent
	AccountID string
	Amount    decimal.Decimal
	Balance   decimal.Decimal
}

func (e *FundsWithdrawnEvent) EventType() string { return "FundsWithdrawn" }

// FundsFrozenEvent 资金冻结事件
type FundsFrozenEvent struct {
	eventsourcing.BaseEvent
	AccountID string
	Amount    decimal.Decimal
	Reason    string
}

func (e *FundsFrozenEvent) EventType() string { return "FundsFrozen" }

// FundsUnfrozenEvent 资金解冻事件
type FundsUnfrozenEvent struct {
	eventsourcing.BaseEvent
	AccountID string
	Amount    decimal.Decimal
}

func (e *FundsUnfrozenEvent) EventType() string { return "FundsUnfrozen" }

// FrozenFundsDeductedEvent 冻结资金扣除事件 (成交)
type FrozenFundsDeductedEvent struct {
	eventsourcing.BaseEvent
	AccountID string
	Amount    decimal.Decimal
}

func (e *FrozenFundsDeductedEvent) EventType() string { return "FrozenFundsDeducted" }
