package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

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
