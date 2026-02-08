package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/eventsourcing"
)

// AccountType 账户类型
type AccountType string

const (
	AccountTypeSpot   AccountType = "SPOT"
	AccountTypeMargin AccountType = "MARGIN"
)

// Account 账户聚合根
type Account struct {
	eventsourcing.AggregateRoot
	ID               uint
	CreatedAt        time.Time
	UpdatedAt        time.Time
	AccountID        string          `gorm:"column:account_id;type:varchar(32);uniqueIndex;not null;comment:账户ID"`
	UserID           string          `gorm:"column:user_id;type:varchar(32);index;not null;comment:用户ID"`
	AccountType      AccountType     `gorm:"column:account_type;type:varchar(20);not null;comment:账户类型"`
	Currency         string          `gorm:"column:currency;type:varchar(10);not null;comment:币种"`
	Balance          decimal.Decimal `gorm:"column:balance;type:decimal(32,18);default:0;not null;comment:总余额"`
	AvailableBalance decimal.Decimal `gorm:"column:available_balance;type:decimal(32,18);default:0;not null;comment:可用余额"`
	FrozenBalance    decimal.Decimal `gorm:"column:frozen_balance;type:decimal(32,18);default:0;not null;comment:冻结余额"`
	BorrowedAmount   decimal.Decimal `gorm:"column:borrowed_amount;type:decimal(32,18);default:0;not null;comment:借款金额"`
	LockedCollateral decimal.Decimal `gorm:"column:locked_collateral;type:decimal(32,18);default:0;not null;comment:锁定质押物"`
	AccruedInterest  decimal.Decimal `gorm:"column:accrued_interest;type:decimal(32,18);default:0;not null;comment:累计利息"`
	VIPLevel         int             `gorm:"column:vip_level;type:int;default:0;not null;comment:VIP等级"`
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
		BorrowedAmount:   decimal.Zero,
		LockedCollateral: decimal.Zero,
		AccruedInterest:  decimal.Zero,
	}
	a.SetID(accountID)

	a.ApplyChange(&AccountCreatedEvent{
		AccountID:   accountID,
		UserID:      userID,
		AccountType: string(accType),
		Currency:    currency,
		VIPLevel:    0,
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
		a.VIPLevel = e.VIPLevel
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
	case *MarginFundsBorrowedEvent:
		a.Balance = e.Balance
		a.AvailableBalance = a.AvailableBalance.Add(e.Amount)
		a.BorrowedAmount = a.BorrowedAmount.Add(e.Amount)
	case *MarginFundsRepaidEvent:
		a.Balance = e.Balance
		a.AvailableBalance = a.AvailableBalance.Sub(e.Amount)
		a.BorrowedAmount = a.BorrowedAmount.Sub(e.Amount)
	case *InterestAccruedEvent:
		a.AccruedInterest = e.Total
	case *InterestSettledEvent:
		a.Balance = e.Balance
		a.BorrowedAmount = a.BorrowedAmount.Add(e.PrincipalDelta)
		a.AccruedInterest = decimal.Zero
	case *VIPLevelUpdatedEvent:
		a.VIPLevel = e.NewLevel
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

// Borrow 借款 (仅限杠杆账户)
func (a *Account) Borrow(amount decimal.Decimal) bool {
	if a.AccountType != AccountTypeMargin {
		return false
	}
	a.ApplyChange(&MarginFundsBorrowedEvent{
		AccountID: a.AccountID,
		Amount:    amount,
		Balance:   a.Balance.Add(amount),
	})
	return true
}

// Repay 还款
func (a *Account) Repay(amount decimal.Decimal) bool {
	// 优先偿还本金，剩余偿还利息（逻辑可根据业务调整）
	if a.AvailableBalance.GreaterThanOrEqual(amount) {
		a.ApplyChange(&MarginFundsRepaidEvent{
			AccountID: a.AccountID,
			Amount:    amount,
			Balance:   a.Balance.Sub(amount),
		})
		return true
	}
	return false
}

// AccrueInterest 计息
func (a *Account) AccrueInterest(rate decimal.Decimal) {
	if a.BorrowedAmount.IsPositive() {
		interest := a.BorrowedAmount.Mul(rate)
		a.ApplyChange(&InterestAccruedEvent{
			AccountID: a.AccountID,
			Amount:    interest,
			Total:     a.AccruedInterest.Add(interest),
		})
	}
}

// SettleInterest 结转利息 (将累计利息资本化或从余额扣除)
func (a *Account) SettleInterest() {
	if a.AccruedInterest.IsZero() {
		return
	}

	// 逻辑：如果是保证金账户，利息计入借款本金；如果是普通账户，利息从余额扣除
	var balDelta, principalDelta decimal.Decimal
	if a.AccountType == AccountTypeMargin {
		principalDelta = a.AccruedInterest
	} else {
		balDelta = a.AccruedInterest.Neg()
	}

	a.ApplyChange(&InterestSettledEvent{
		AccountID:      a.AccountID,
		Amount:         a.AccruedInterest,
		Balance:        a.Balance.Add(balDelta),
		PrincipalDelta: principalDelta,
	})
}

// UpdateVIPLevel 更新 VIP 等级
func (a *Account) UpdateVIPLevel(level int) {
	if a.VIPLevel != level {
		a.ApplyChange(&VIPLevelUpdatedEvent{
			AccountID: a.AccountID,
			OldLevel:  a.VIPLevel,
			NewLevel:  level,
		})
	}
}
