// Package domain 资金服务领域层
// 生成摘要：
// 1) 定义资金账户聚合根
// 2) 定义资金流水实体
// 3) 实现冻结、解冻、扣减、增加的领域逻辑（不仅是CRUD）
package domain

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AccountType 账户类型
type AccountType int8

const (
	AccountTypeUser     AccountType = 1 // 用户资金账户
	AccountTypeMerchant AccountType = 2 // 商家资金账户
	AccountTypePlatform AccountType = 3 // 平台运营账户
	AccountTypeMargin   AccountType = 4 // 保证金账户
)

// Currency 货币类型
type Currency int8

const (
	CurrencyCNY Currency = 1
	CurrencyUSD Currency = 2
)

// AccountStatus 账户状态
type AccountStatus int8

const (
	AccountStatusActive AccountStatus = 1 // 正常
	AccountStatusFrozen AccountStatus = 2 // 冻结（全账户）
	AccountStatusClosed AccountStatus = 3 //以此
)

// Account 资金账户聚合根
type Account struct {
	gorm.Model
	OwnerID  uint64        `gorm:"column:owner_id;index;not null"`
	Type     AccountType   `gorm:"column:type;type:tinyint;not null"`
	Currency Currency      `gorm:"column:currency;type:tinyint;not null"`
	Status   AccountStatus `gorm:"column:status;type:tinyint;not null;default:1"`

	Balance   int64 `gorm:"column:balance;not null;default:0"`   // 总余额(分)
	Available int64 `gorm:"column:available;not null;default:0"` // 可用余额(分)
	Frozen    int64 `gorm:"column:frozen;not null;default:0"`    // 冻结余额(分)

	// 乐观锁版本号
	Version int64 `gorm:"column:version;not null;default:0"`

	// 领域事件
	domainEvents []DomainEvent `gorm:"-"`
}

// TableName 表名
func (Account) TableName() string {
	return "accounts"
}

// TransactionType 交易类型
type TransactionType int8

const (
	TransactionTypeDeposit     TransactionType = 1 // 充值
	TransactionTypeWithdraw    TransactionType = 2 // 提现
	TransactionTypeTransferIn  TransactionType = 3 // 转入
	TransactionTypeTransferOut TransactionType = 4 // 转出
	TransactionTypePayment     TransactionType = 5 // 支付
	TransactionTypeRefund      TransactionType = 6 // 退款
	TransactionTypeFreeze      TransactionType = 7 // 冻结
	TransactionTypeUnfreeze    TransactionType = 8 // 解冻
	TransactionTypeDeduct      TransactionType = 9 // 扣减
)

// Transaction 资金流水
type Transaction struct {
	gorm.Model
	TransactionID string          `gorm:"column:transaction_id;type:varchar(32);unique_index;not null"`
	AccountID     uint            `gorm:"column:account_id;index;not null"`
	Type          TransactionType `gorm:"column:type;type:tinyint;not null"`
	Amount        int64           `gorm:"column:amount;not null"`        // 变动金额
	BalanceAfter  int64           `gorm:"column:balance_after;not null"` // 变动后余额（这里的余额指引起变动的那个余额，如可用或冻结）
	ReferenceID   string          `gorm:"column:reference_id;type:varchar(64);index"`
	Remark        string          `gorm:"column:remark;type:varchar(255)"`
}

// TableName 表名
func (Transaction) TableName() string {
	return "transactions"
}

// NewAccount 创建资金账户
func NewAccount(ownerID uint64, accType AccountType, currency Currency) *Account {
	return &Account{
		OwnerID:   ownerID,
		Type:      accType,
		Currency:  currency,
		Status:    AccountStatusActive,
		Balance:   0,
		Available: 0,
		Frozen:    0,
		Version:   1,
	}
}

// Deposit 增加资金（充值/入金）
func (a *Account) Deposit(amount int64, refID, source string) (*Transaction, error) {
	if a.Status != AccountStatusActive {
		return nil, errors.New("account is not active")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	a.Balance += amount
	a.Available += amount
	// 乐观锁控制通过 Repository 层实现，这里只变更内存状态

	now := time.Now()
	txID := fmt.Sprintf("TX%s%04d", now.Format("20060102150405"), now.UnixNano()%10000)

	tx := &Transaction{
		TransactionID: txID,
		AccountID:     a.ID,
		Type:          TransactionTypeDeposit,
		Amount:        amount,
		BalanceAfter:  a.Balance,
		ReferenceID:   refID,
		Remark:        source,
	}

	a.addEvent(&FundsDepositedEvent{
		AccountID: uint64(a.ID),
		Amount:    amount,
		Balance:   a.Balance,
		Timestamp: now,
	})

	return tx, nil
}

// Freeze 冻结资金
func (a *Account) Freeze(amount int64, refID, reason string) (*Transaction, error) {
	if a.Status != AccountStatusActive {
		return nil, errors.New("account is not active")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	if a.Available < amount {
		return nil, errors.New("insufficient available balance")
	}

	a.Available -= amount
	a.Frozen += amount

	now := time.Now()
	txID := fmt.Sprintf("TX%s%04d", now.Format("20060102150405"), now.UnixNano()%10000)

	tx := &Transaction{
		TransactionID: txID,
		AccountID:     a.ID,
		Type:          TransactionTypeFreeze,
		Amount:        amount,
		BalanceAfter:  a.Frozen,
		ReferenceID:   refID,
		Remark:        reason,
	}

	a.addEvent(&FundsFrozenEvent{
		AccountID: uint64(a.ID),
		Amount:    amount,
		Frozen:    a.Frozen,
		Reason:    reason,
		Timestamp: now,
	})

	return tx, nil
}

// Unfreeze 解冻资金
func (a *Account) Unfreeze(amount int64, refID, reason string) (*Transaction, error) {
	if a.Status != AccountStatusActive {
		return nil, errors.New("account is not active")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	if a.Frozen < amount {
		return nil, errors.New("insufficient frozen balance")
	}

	a.Frozen -= amount
	a.Available += amount

	now := time.Now()
	txID := fmt.Sprintf("TX%s%04d", now.Format("20060102150405"), now.UnixNano()%10000)

	tx := &Transaction{
		TransactionID: txID,
		AccountID:     a.ID,
		Type:          TransactionTypeUnfreeze,
		Amount:        amount,
		BalanceAfter:  a.Frozen,
		ReferenceID:   refID,
		Remark:        reason,
	}

	a.addEvent(&FundsUnfrozenEvent{
		AccountID: uint64(a.ID),
		Amount:    amount,
		Frozen:    a.Frozen,
		Reason:    reason,
		Timestamp: now,
	})

	return tx, nil
}

// Deduct 扣减资金（直接扣减可用，或扣减冻结）
func (a *Account) Deduct(amount int64, fromFrozen bool, refID, reason string) (*Transaction, error) {
	if a.Status != AccountStatusActive {
		return nil, errors.New("account is not active")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	if fromFrozen {
		if a.Frozen < amount {
			return nil, errors.New("insufficient frozen balance")
		}
		a.Frozen -= amount
	} else {
		if a.Available < amount {
			return nil, errors.New("insufficient available balance")
		}
		a.Available -= amount
	}
	a.Balance -= amount

	now := time.Now()
	txID := fmt.Sprintf("TX%s%04d", now.Format("20060102150405"), now.UnixNano()%10000)

	tx := &Transaction{
		TransactionID: txID,
		AccountID:     a.ID,
		Type:          TransactionTypeDeduct,
		Amount:        -amount,
		BalanceAfter:  a.Balance,
		ReferenceID:   refID,
		Remark:        reason,
	}

	a.addEvent(&FundsDeductedEvent{
		AccountID: uint64(a.ID),
		Amount:    amount,
		Balance:   a.Balance,
		Reason:    reason,
		Timestamp: now,
	})

	return tx, nil
}

func (a *Account) addEvent(event DomainEvent) {
	a.domainEvents = append(a.domainEvents, event)
}

func (a *Account) GetDomainEvents() []DomainEvent {
	return a.domainEvents
}

func (a *Account) ClearDomainEvents() {
	a.domainEvents = nil
}
