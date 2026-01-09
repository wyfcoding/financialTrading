package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// PositionRepository 持仓仓储接口
type PositionRepository interface {
	// Save 保存或更新持仓
	Save(ctx context.Context, position *Position) error
	// Get 根据持仓 ID 获取持仓
	Get(ctx context.Context, positionID string) (*Position, error)
	// GetByUser 获取用户持仓列表
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]*Position, int64, error)
	// GetBySymbol 获取交易对持仓列表
	GetBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*Position, int64, error)
	// Update 显式更新持仓全量信息
	Update(ctx context.Context, position *Position) error
	// Close 平仓并记录平仓价格
	Close(ctx context.Context, positionID string, closePrice decimal.Decimal) error

	// ExecWithBarrier 在分布式事务屏障下执行业务逻辑
	ExecWithBarrier(ctx context.Context, barrier any, fn func(ctx context.Context) error) error
}
