package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// AccountType 账户类型
type AccountType string

const (
	AccountTypeSpot   AccountType = "SPOT"
	AccountTypeMargin AccountType = "MARGIN"
)

// Account 账户聚合根
// 纯领域模型，不依赖数据库标签
type Account struct {
	ID               string
	UserID           string
	AccountType      AccountType
	Currency         string
	Balance          decimal.Decimal
	AvailableBalance decimal.Decimal
	FrozenBalance    decimal.Decimal
	Version          int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NewAccount 创建新账户
func NewAccount(id, userID, currency string, accType AccountType) *Account {
	return &Account{
		ID:               id,
		UserID:           userID,
		AccountType:      accType,
		Currency:         currency,
		Balance:          decimal.Zero,
		AvailableBalance: decimal.Zero,
		FrozenBalance:    decimal.Zero,
		Version:          0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// Deposit 充值
func (a *Account) Deposit(amount decimal.Decimal) {
	if amount.IsPositive() {
		a.Balance = a.Balance.Add(amount)
		a.AvailableBalance = a.AvailableBalance.Add(amount)
		a.UpdatedAt = time.Now()
	}
}

// Freeze 冻结
func (a *Account) Freeze(amount decimal.Decimal) bool {
	if a.AvailableBalance.GreaterThanOrEqual(amount) {
		a.AvailableBalance = a.AvailableBalance.Sub(amount)
		a.FrozenBalance = a.FrozenBalance.Add(amount)
		a.UpdatedAt = time.Now()
		return true
	}
	return false
}

// Unfreeze 解冻
func (a *Account) Unfreeze(amount decimal.Decimal) bool {
	if a.FrozenBalance.GreaterThanOrEqual(amount) {
		a.FrozenBalance = a.FrozenBalance.Sub(amount)
		a.AvailableBalance = a.AvailableBalance.Add(amount)
		a.UpdatedAt = time.Now()
		return true
	}
	return false
}

// DeductFrozen 扣除冻结（例如成交）
func (a *Account) DeductFrozen(amount decimal.Decimal) bool {
	if a.FrozenBalance.GreaterThanOrEqual(amount) {
		a.Balance = a.Balance.Sub(amount)
		a.FrozenBalance = a.FrozenBalance.Sub(amount)
		a.UpdatedAt = time.Now()
		return true
	}
	return false
}

// Withdraw 提现（直接扣减可用）
func (a *Account) Withdraw(amount decimal.Decimal) bool {
	if a.AvailableBalance.GreaterThanOrEqual(amount) {
		a.Balance = a.Balance.Sub(amount)
		a.AvailableBalance = a.AvailableBalance.Sub(amount)
		a.UpdatedAt = time.Now()
		return true
	}
	return false
}
