// Package domain 包含执行服务的领域模型
package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "PENDING"
	ExecutionStatusExecuting ExecutionStatus = "EXECUTING"
	ExecutionStatusCompleted ExecutionStatus = "COMPLETED"
	ExecutionStatusFailed    ExecutionStatus = "FAILED"
)

// Execution 执行记录实体
type Execution struct {
	// 执行 ID
	ExecutionID string
	// 订单 ID
	OrderID string
	// 用户 ID
	UserID string
	// 交易对
	Symbol string
	// 买卖方向
	Side string
	// 执行价格
	ExecutedPrice decimal.Decimal
	// 执行数量
	ExecutedQuantity decimal.Decimal
	// 执行状态
	Status ExecutionStatus
	// 创建时间
	CreatedAt time.Time
	// 更新时间
	UpdatedAt time.Time
}

// ExecutionRepository 执行记录仓储接口
type ExecutionRepository interface {
	// 保存执行记录
	Save(ctx context.Context, execution *Execution) error
	// 获取执行记录
	Get(ctx context.Context, executionID string) (*Execution, error)
	// 获取订单执行历史
	GetByOrder(ctx context.Context, orderID string) ([]*Execution, error)
	// 获取用户执行历史
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]*Execution, int64, error)
}
