// 包 domain 账户服务的领域模型
package domain

import (
	"context"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Account 账户实体
// 代表用户的资金账户，包含余额和状态信息
type Account struct {
	gorm.Model
	// 账户 ID (业务主键)，全局唯一
	AccountID string `gorm:"column:account_id;type:varchar(32);uniqueIndex;not null" json:"account_id"`
	// 用户 ID，关联的用户
	UserID string `gorm:"column:user_id;type:varchar(32);index;not null" json:"user_id"`
	// 账户类型（SPOT: 现货, MARGIN: 杠杆, FUTURES: 合约）
	AccountType string `gorm:"column:account_type;type:varchar(20);not null" json:"account_type"`
	// 货币（如 USD, BTC, ETH）
	Currency string `gorm:"column:currency;type:varchar(10);not null" json:"currency"`
	// 总余额 = 可用余额 + 冻结余额
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
	// Save 保存或更新账户
	Save(ctx context.Context, account *Account) error
	// Get 根据账户 ID 获取账户
	Get(ctx context.Context, accountID string) (*Account, error)
	// GetByUser 根据用户 ID 获取账户列表
	GetByUser(ctx context.Context, userID string) ([]*Account, error)
	// UpdateBalance 显式更新账户余额（含锁或原子操作建议在仓储实现中处理）
	UpdateBalance(ctx context.Context, accountID string, balance, availableBalance, frozenBalance decimal.Decimal) error
}

// TransactionRepository 交易记录仓储接口
type TransactionRepository interface {
	// Save 保存交易记录
	Save(ctx context.Context, transaction *Transaction) error
	// GetHistory 获取交易历史分页列表
	GetHistory(ctx context.Context, accountID string, limit, offset int) ([]*Transaction, int64, error)
}
