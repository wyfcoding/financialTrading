package domain

import (
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/eventsourcing"
	"gorm.io/gorm"
)

// AccountType 账户类型
type AccountType string

const (
	AccountTypeSpot   AccountType = "SPOT"
	AccountTypeMargin AccountType = "MARGIN"
)

// Account 账户聚合根
type Account struct {
	gorm.Model
	eventsourcing.AggregateRoot
	AccountID        string          `gorm:"column:account_id;type:varchar(32);uniqueIndex;not null;comment:账户ID"`
	UserID           string          `gorm:"column:user_id;type:varchar(32);index;not null;comment:用户ID"`
	AccountType      AccountType     `gorm:"column:account_type;type:varchar(20);not null;comment:账户类型"`
	Currency         string          `gorm:"column:currency;type:varchar(10);not null;comment:币种"`
	Balance          decimal.Decimal `gorm:"column:balance;type:decimal(32,18);default:0;not null;comment:总余额"`
	AvailableBalance decimal.Decimal `gorm:"column:available_balance;type:decimal(32,18);default:0;not null;comment:可用余额"`
	FrozenBalance    decimal.Decimal `gorm:"column:frozen_balance;type:decimal(32,18);default:0;not null;comment:冻结余额"`
}

func (Account) TableName() string {
	return "accounts"
}

// NewAccount 创建新账户
func NewAccount(accountID, userID, currency string, accType AccountType) *Account {
	a := &Account{
		AccountID:        accountID,
		UserID:           userID,
		AccountType:      accType,
		Currency:         currency,
		Balance:          decimal.Zero,
		AvailableBalance: decimal.Zero,
		FrozenBalance:    decimal.Zero,
	}
	a.SetID(accountID)

	a.ApplyChange(&AccountCreatedEvent{
		AccountID:   accountID,
		UserID:      userID,
		AccountType: string(accType),
		Currency:    currency,
	})
	return a
}

// Apply 实现了 eventsourcing.EventApplier 接口
func (a *Account) Apply(event eventsourcing.DomainEvent) {
	switch e := event.(type) {
	case *AccountCreatedEvent:
		a.AccountID = e.AccountID
		a.UserID = e.UserID
		a.AccountType = AccountType(e.AccountType)
		a.Currency = e.Currency
	case *FundsDepositedEvent:
		a.Balance = e.Balance
		a.AvailableBalance = a.AvailableBalance.Add(e.Amount)
	case *FundsWithdrawnEvent:
		a.Balance = e.Balance
		a.AvailableBalance = a.AvailableBalance.Sub(e.Amount)
	case *FundsFrozenEvent:
		a.AvailableBalance = a.AvailableBalance.Sub(e.Amount)
		a.FrozenBalance = a.FrozenBalance.Add(e.Amount)
	case *FundsUnfrozenEvent:
		a.AvailableBalance = a.AvailableBalance.Add(e.Amount)
		a.FrozenBalance = a.FrozenBalance.Sub(e.Amount)
	case *FrozenFundsDeductedEvent:
		a.Balance = a.Balance.Sub(e.Amount)
		a.FrozenBalance = a.FrozenBalance.Sub(e.Amount)
	}
}

// Deposit 充值
func (a *Account) Deposit(amount decimal.Decimal) {
	if amount.IsPositive() {
		a.ApplyChange(&FundsDepositedEvent{
			AccountID: a.AccountID,
			Amount:    amount,
			Balance:   a.Balance.Add(amount),
		})
	}
}

// Freeze 冻结
func (a *Account) Freeze(amount decimal.Decimal, reason string) bool {
	if a.AvailableBalance.GreaterThanOrEqual(amount) {
		a.ApplyChange(&FundsFrozenEvent{
			AccountID: a.AccountID,
			Amount:    amount,
			Reason:    reason,
		})
		return true
	}
	return false
}

// Unfreeze 解冻
func (a *Account) Unfreeze(amount decimal.Decimal) bool {
	if a.FrozenBalance.GreaterThanOrEqual(amount) {
		a.ApplyChange(&FundsUnfrozenEvent{
			AccountID: a.AccountID,
			Amount:    amount,
		})
		return true
	}
	return false
}

// DeductFrozen 扣除冻结（例如成交）
func (a *Account) DeductFrozen(amount decimal.Decimal) bool {
	if a.FrozenBalance.GreaterThanOrEqual(amount) {
		a.ApplyChange(&FrozenFundsDeductedEvent{
			AccountID: a.AccountID,
			Amount:    amount,
		})
		return true
	}
	return false
}

// Withdraw 提现（直接扣减可用）
func (a *Account) Withdraw(amount decimal.Decimal) bool {
	if a.AvailableBalance.GreaterThanOrEqual(amount) {
		a.ApplyChange(&FundsWithdrawnEvent{
			AccountID: a.AccountID,
			Amount:    amount,
			Balance:   a.Balance.Sub(amount),
		})
		return true
	}
	return false
}
