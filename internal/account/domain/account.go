// Package domain 包含账户服务的领域模型
package domain

import (
	"context"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Account 账户实体
type Account struct {
	gorm.Model
	// 账户 ID (业务主键)
	AccountID string `gorm:"column:account_id;type:varchar(32);uniqueIndex;not null" json:"account_id"`
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(32);index;not null" json:"user_id"`
	// 账户类型（SPOT, MARGIN, FUTURES）
	AccountType string `gorm:"column:account_type;type:varchar(20);not null" json:"account_type"`
	// 货币
	Currency string `gorm:"column:currency;type:varchar(10);not null" json:"currency"`
	// 余额
	Balance decimal.Decimal `gorm:"column:balance;type:decimal(32,18);default:0;not null" json:"balance"`
	// 可用余额
	AvailableBalance decimal.Decimal `gorm:"column:available_balance;type:decimal(32,18);default:0;not null" json:"available_balance"`
	// 冻结余额
	FrozenBalance decimal.Decimal `gorm:"column:frozen_balance;type:decimal(32,18);default:0;not null" json:"frozen_balance"`
}

// Transaction 交易记录
type Transaction struct {
	gorm.Model
	// 交易 ID (业务主键)
	TransactionID string `gorm:"column:transaction_id;type:varchar(32);uniqueIndex;not null" json:"transaction_id"`
	// 账户 ID
	AccountID string `gorm:"column:account_id;type:varchar(32);index;not null" json:"account_id"`
	// 交易类型（DEPOSIT, WITHDRAW, TRADE, FEE）
	Type string `gorm:"column:type;type:varchar(20);not null" json:"type"`
	// 金额
	Amount decimal.Decimal `gorm:"column:amount;type:decimal(32,18);not null" json:"amount"`
	// 状态
	Status string `gorm:"column:status;type:varchar(20);not null" json:"status"`
}

// AccountRepository 账户仓储接口
type AccountRepository interface {
	// 保存账户
	Save(ctx context.Context, account *Account) error
	// 获取账户
	Get(ctx context.Context, accountID string) (*Account, error)
	// 获取用户账户
	GetByUser(ctx context.Context, userID string) ([]*Account, error)
	// 更新余额
	UpdateBalance(ctx context.Context, accountID string, balance, availableBalance, frozenBalance decimal.Decimal) error
}

// TransactionRepository 交易记录仓储接口
type TransactionRepository interface {
	// 保存交易记录
	Save(ctx context.Context, transaction *Transaction) error
	// 获取交易历史
	GetHistory(ctx context.Context, accountID string, limit, offset int) ([]*Transaction, int64, error)
}
