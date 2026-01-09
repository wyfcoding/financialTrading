package domain

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
)

var ErrConcurrentUpdate = errors.New("account record concurrent update detected")

// AccountRepository 账户仓储接口
type AccountRepository interface {
	// Save 保存或更新账户
	Save(ctx context.Context, account *Account) error
	// Get 根据账户 ID 获取账户
	Get(ctx context.Context, accountID string) (*Account, error)
	// GetByUser 根据用户 ID 获取账户列表
	GetByUser(ctx context.Context, userID string) ([]*Account, error)
	// UpdateBalance 显式更新账户余额 (带乐观锁版本检查)
	UpdateBalance(ctx context.Context, accountID string, balance, availableBalance, frozenBalance decimal.Decimal, currentVersion int64) error

	// ExecWithBarrier 在分布式事务屏障下执行业务逻辑
	// barrier 类型应为 *dtmgrpc.BranchBarrier，使用 any 避免领域层强依赖
	ExecWithBarrier(ctx context.Context, barrier any, fn func(ctx context.Context) error) error
}

// TransactionRepository 交易记录仓储接口
type TransactionRepository interface {
	// Save 保存交易记录
	Save(ctx context.Context, transaction *Transaction) error
	// GetHistory 获取交易历史分页列表
	GetHistory(ctx context.Context, accountID string, limit, offset int) ([]*Transaction, int64, error)
}
