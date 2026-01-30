package domain

import (
	"github.com/shopspring/decimal"
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
	AccountID        string          `gorm:"column:account_id;type:varchar(32);uniqueIndex;not null;comment:账户ID"`
	UserID           string          `gorm:"column:user_id;type:varchar(32);index;not null;comment:用户ID"`
	AccountType      AccountType     `gorm:"column:account_type;type:varchar(20);not null;comment:账户类型"`
	Currency         string          `gorm:"column:currency;type:varchar(10);not null;comment:币种"`
	Balance          decimal.Decimal `gorm:"column:balance;type:decimal(32,18);default:0;not null;comment:总余额"`
	AvailableBalance decimal.Decimal `gorm:"column:available_balance;type:decimal(32,18);default:0;not null;comment:可用余额"`
	FrozenBalance    decimal.Decimal `gorm:"column:frozen_balance;type:decimal(32,18);default:0;not null;comment:冻结余额"`
	Version          int64           `gorm:"column:version;default:0;not null;comment:乐观锁版本"`
}

func (Account) TableName() string {
	return "accounts"
}

// NewAccount 创建新账户
func NewAccount(accountID, userID, currency string, accType AccountType) *Account {
	return &Account{
		AccountID:        accountID,
		UserID:           userID,
		AccountType:      accType,
		Currency:         currency,
		Balance:          decimal.Zero,
		AvailableBalance: decimal.Zero,
		FrozenBalance:    decimal.Zero,
		Version:          0,
	}
}

// Deposit 充值
func (a *Account) Deposit(amount decimal.Decimal) {
	if amount.IsPositive() {
		a.Balance = a.Balance.Add(amount)
		a.AvailableBalance = a.AvailableBalance.Add(amount)
	}
}

// Freeze 冻结
func (a *Account) Freeze(amount decimal.Decimal) bool {
	if a.AvailableBalance.GreaterThanOrEqual(amount) {
		a.AvailableBalance = a.AvailableBalance.Sub(amount)
		a.FrozenBalance = a.FrozenBalance.Add(amount)
		return true
	}
	return false
}

// Unfreeze 解冻
func (a *Account) Unfreeze(amount decimal.Decimal) bool {
	if a.FrozenBalance.GreaterThanOrEqual(amount) {
		a.FrozenBalance = a.FrozenBalance.Sub(amount)
		a.AvailableBalance = a.AvailableBalance.Add(amount)
		return true
	}
	return false
}

// DeductFrozen 扣除冻结（例如成交）
func (a *Account) DeductFrozen(amount decimal.Decimal) bool {
	if a.FrozenBalance.GreaterThanOrEqual(amount) {
		a.Balance = a.Balance.Sub(amount)
		a.FrozenBalance = a.FrozenBalance.Sub(amount)
		return true
	}
	return false
}

// Withdraw 提现（直接扣减可用）
func (a *Account) Withdraw(amount decimal.Decimal) bool {
	if a.AvailableBalance.GreaterThanOrEqual(amount) {
		a.Balance = a.Balance.Sub(amount)
		a.AvailableBalance = a.AvailableBalance.Sub(amount)
		return true
	}
	return false
}
