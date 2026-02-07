// 生成摘要：充值和提现仓储接口定义。
package domain

import "context"

// DepositOrderRepository 充值订单仓储接口
type DepositOrderRepository interface {
	// Save 保存充值订单
	Save(ctx context.Context, deposit *DepositOrder) error
	// FindByID 根据ID查询
	FindByID(ctx context.Context, id uint) (*DepositOrder, error)
	// FindByDepositNo 根据充值单号查询
	FindByDepositNo(ctx context.Context, depositNo string) (*DepositOrder, error)
	// FindByUserID 根据用户ID查询列表
	FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*DepositOrder, int64, error)
	// FindByAccountID 根据账户ID查询列表
	FindByAccountID(ctx context.Context, accountID string, offset, limit int) ([]*DepositOrder, int64, error)
	// Update 更新充值订单
	Update(ctx context.Context, deposit *DepositOrder) error
	// WithTx 事务执行
	WithTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

// WithdrawalOrderRepository 提现订单仓储接口
type WithdrawalOrderRepository interface {
	// Save 保存提现订单
	Save(ctx context.Context, withdrawal *WithdrawalOrder) error
	// FindByID 根据ID查询
	FindByID(ctx context.Context, id uint) (*WithdrawalOrder, error)
	// FindByWithdrawalNo 根据提现单号查询
	FindByWithdrawalNo(ctx context.Context, withdrawalNo string) (*WithdrawalOrder, error)
	// FindByUserID 根据用户ID查询列表
	FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*WithdrawalOrder, int64, error)
	// FindByAccountID 根据账户ID查询列表
	FindByAccountID(ctx context.Context, accountID string, offset, limit int) ([]*WithdrawalOrder, int64, error)
	// FindPendingForAudit 查询待审核列表
	FindPendingForAudit(ctx context.Context, offset, limit int) ([]*WithdrawalOrder, int64, error)
	// Update 更新提现订单
	Update(ctx context.Context, withdrawal *WithdrawalOrder) error
	// WithTx 事务执行
	WithTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
