// Package domain 资金池与银行账户领域模型
// 生成摘要：
// 1) 定义 BankAccount（实体）：代表外部实体银行账户
// 2) 定义 CashPool（聚合根）：逻辑上的资金池，聚合多个物理账户
// 3) 实现资金池水位监控与预警逻辑
package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrInsufficientPoolBalance = errors.New("insufficient pool balance")
	ErrAccountAlreadyExists    = errors.New("bank account already exists")
)

// BankAccountStatus 银行账户状态
type BankAccountStatus string

const (
	BankAccountStatusActive   BankAccountStatus = "ACTIVE"
	BankAccountStatusInactive BankAccountStatus = "INACTIVE"
	BankAccountStatusFrozen   BankAccountStatus = "FROZEN"
)

// BankAccount 外部实体银行账户
type BankAccount struct {
	gorm.Model
	PoolID        uint64            `gorm:"column:pool_id;index;not null"` // 所属资金池
	BankName      string            `gorm:"column:bank_name;type:varchar(128);not null"`
	AccountNo     string            `gorm:"column:account_no;type:varchar(64);uniqueIndex;not null"` // 银行账号
	AccountName   string            `gorm:"column:account_name;type:varchar(128);not null"`
	SwiftCode     string            `gorm:"column:swift_code;type:varchar(32)"`
	Currency      string            `gorm:"column:currency;type:char(3);not null"`
	Balance       decimal.Decimal   `gorm:"column:balance;type:decimal(20,4);not null"` // 账面余额
	Status        BankAccountStatus `gorm:"column:status;type:varchar(32);not null;default:'ACTIVE'"`
	LastCheckTime *time.Time        `gorm:"column:last_check_time"` // 上次余额核对时间
	Attributes    map[string]string `gorm:"column:attributes;serializer:json"`
}

func (BankAccount) TableName() string { return "treasury_bank_accounts" }

// CashPool 资金池（聚合根）
type CashPool struct {
	gorm.Model
	Name         string          `gorm:"column:name;type:varchar(128);uniqueIndex;not null"`
	Currency     string          `gorm:"column:currency;type:char(3);not null"`
	TotalBalance decimal.Decimal `gorm:"column:total_balance;type:decimal(20,4);not null"` // 汇总余额
	MinTarget    decimal.Decimal `gorm:"column:min_target;type:decimal(20,4);not null"`    // 最低流动性目标
	MaxTarget    decimal.Decimal `gorm:"column:max_target;type:decimal(20,4);not null"`    // 最高流动性目标
	Accounts     []BankAccount   `gorm:"foreignKey:PoolID"`                                // 包含的物理账户
	ManagerID    string          `gorm:"column:manager_id;type:varchar(64)"`               // 负责人
	Description  string          `gorm:"column:description;type:text"`
}

func (CashPool) TableName() string { return "treasury_cash_pools" }

// NewCashPool 创建资金池
func NewCashPool(name, currency string, min, max decimal.Decimal) *CashPool {
	return &CashPool{
		Name:         name,
		Currency:     currency,
		TotalBalance: decimal.Zero,
		MinTarget:    min,
		MaxTarget:    max,
		Accounts:     make([]BankAccount, 0),
	}
}

// AddAccount 添加物理账户到资金池
func (p *CashPool) AddAccount(account BankAccount) error {
	if account.Currency != p.Currency {
		return errors.New("currency mismatch")
	}
	p.Accounts = append(p.Accounts, account)
	p.RecalculateBalance()
	return nil
}

// RecalculateBalance 重新计算资金池总余额
func (p *CashPool) RecalculateBalance() {
	total := decimal.Zero
	for _, acc := range p.Accounts {
		if acc.Status == BankAccountStatusActive {
			total = total.Add(acc.Balance)
		}
	}
	p.TotalBalance = total
}

// CheckLiquidity 检查流动性水位
// 返回: (isLow, isHigh, deviation)
func (p *CashPool) CheckLiquidity() (bool, bool, decimal.Decimal) {
	if p.TotalBalance.LessThan(p.MinTarget) {
		return true, false, p.MinTarget.Sub(p.TotalBalance)
	}
	if p.TotalBalance.GreaterThan(p.MaxTarget) {
		return false, true, p.TotalBalance.Sub(p.MaxTarget)
	}
	return false, false, decimal.Zero
}

// UpdateAccountBalance 更新物理账户余额并联动更新资金池
func (p *CashPool) UpdateAccountBalance(accountID uint, newBalance decimal.Decimal) bool {
	updated := false
	for i := range p.Accounts {
		if p.Accounts[i].ID == accountID {
			p.Accounts[i].Balance = newBalance
			now := time.Now()
			p.Accounts[i].LastCheckTime = &now
			updated = true
			break
		}
	}
	if updated {
		p.RecalculateBalance()
	}
	return updated
}
