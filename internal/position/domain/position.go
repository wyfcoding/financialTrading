// 包 domain 持仓服务的领域模型
package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Position 持仓实体
// 代表用户在某个交易对上的持仓信息
type Position struct {
	gorm.Model
	// 持仓 ID
	PositionID string `gorm:"column:position_id;type:varchar(32);uniqueIndex;not null" json:"position_id"`
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(32);index;not null" json:"user_id"`
	// 交易对
	Symbol string `gorm:"column:symbol;type:varchar(20);index;not null" json:"symbol"`
	// 买卖方向 (LONG/SHORT)
	Side string `gorm:"column:side;type:varchar(10);not null" json:"side"`
	// 持仓数量
	Quantity decimal.Decimal `gorm:"column:quantity;type:decimal(32,18);not null" json:"quantity"`
	// 开仓价格 (平均成本)
	EntryPrice decimal.Decimal `gorm:"column:entry_price;type:decimal(32,18);not null" json:"entry_price"`
	// 当前价格
	CurrentPrice decimal.Decimal `gorm:"column:current_price;type:decimal(32,18);not null" json:"current_price"`
	// 未实现盈亏
	UnrealizedPnL decimal.Decimal `gorm:"column:unrealized_pnl;type:decimal(32,18);not null" json:"unrealized_pnl"`
	// 已实现盈亏
	RealizedPnL decimal.Decimal `gorm:"column:realized_pnl;type:decimal(32,18);not null" json:"realized_pnl"`
	// 开仓时间
	OpenedAt time.Time `gorm:"column:opened_at;type:datetime;not null" json:"opened_at"`
	// 平仓时间
	ClosedAt *time.Time `gorm:"column:closed_at;type:datetime" json:"closed_at"`
	// 状态
	Status string `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
}

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
}
