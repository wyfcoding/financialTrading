// Package domain 包含执行服务的领域模型、仓储接口和领域服务。
// 这是领域驱动设计（DDD）中的核心层，负责表达业务概念、规则和状态。
package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ExecutionStatus 定义了订单执行的状态。
type ExecutionStatus string

// OrderSide 定义了订单的买卖方向。
type OrderSide string

const (
	ExecutionStatusPending   ExecutionStatus = "PENDING"   // 待处理：订单已接收，等待执行
	ExecutionStatusExecuting ExecutionStatus = "EXECUTING" // 执行中：订单正在被撮合或发送到外部交易所
	ExecutionStatusCompleted ExecutionStatus = "COMPLETED" // 已完成：订单已完全成交
	ExecutionStatusFailed    ExecutionStatus = "FAILED"    // 已失败：订单执行失败
	ExecutionStatusCancelled ExecutionStatus = "CANCELLED" // 已取消：订单在成交前被取消
)

const (
	SideBuy  OrderSide = "BUY"  // 买单
	SideSell OrderSide = "SELL" // 卖单
)

// Execution 是执行记录的领域实体（Entity）。
// 它代表一次订单执行（或部分执行）的结果，是系统中的一个关键事实。
type Execution struct {
	gorm.Model // 嵌入 gorm.Model，包含 ID, CreatedAt, UpdatedAt, DeletedAt

	// ExecutionID 是执行记录的唯一标识符，由系统生成。
	ExecutionID string `gorm:"column:execution_id;type:varchar(50);uniqueIndex;not null" json:"execution_id"`
	// OrderID 是关联的订单ID。
	OrderID string `gorm:"column:order_id;type:varchar(50);index;not null" json:"order_id"`
	// UserID 是发起订单的用户的ID。
	UserID string `gorm:"column:user_id;type:varchar(50);index;not null" json:"user_id"`
	// Symbol 是交易对，例如 "BTC/USDT"。
	Symbol string `gorm:"column:symbol;type:varchar(50);not null" json:"symbol"`
	// Side 是交易方向 (BUY 或 SELL)。
	Side OrderSide `gorm:"column:side;type:varchar(10);not null" json:"side"`
	// ExecutedPrice 是本次执行的成交价格。
	ExecutedPrice decimal.Decimal `gorm:"column:executed_price;type:decimal(20,8);not null" json:"executed_price"`
	// ExecutedQuantity 是本次执行的成交数量。
	ExecutedQuantity decimal.Decimal `gorm:"column:executed_quantity;type:decimal(20,8);not null" json:"executed_quantity"`
	// Status 是本次执行的状态。
	Status ExecutionStatus `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
	// CreatedAt 是记录创建时间，由 gorm.Model 提供。
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
	// UpdatedAt 是记录更新时间，由 gorm.Model 提供。
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;not null" json:"updated_at"`
}

// ExecutionRepository 是执行记录的仓储接口（Repository Interface）。
// 它定义了对 Execution 实体的持久化操作，将领域层与具体的数据存储实现解耦。
type ExecutionRepository interface {
	// Save 保存一个新的或更新一个已有的执行记录。
	Save(ctx context.Context, execution *Execution) error

	// Get 根据 ExecutionID 获取一个执行记录。
	Get(ctx context.Context, executionID string) (*Execution, error)

	// GetByOrder 获取指定订单的所有执行记录（一个订单可能分多次执行）。
	GetByOrder(ctx context.Context, orderID string) ([]*Execution, error)

	// GetByUser 分页获取指定用户的执行历史记录。
	// 返回执行记录列表和记录总数。
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]*Execution, int64, error)
}
