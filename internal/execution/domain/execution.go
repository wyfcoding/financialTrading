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
	ExecutionStatusPending   ExecutionStatus = "PENDING"   // 等待执行
	ExecutionStatusExecuting ExecutionStatus = "EXECUTING" // 正在执行
	ExecutionStatusCompleted ExecutionStatus = "COMPLETED" // 执行完成
	ExecutionStatusFailed    ExecutionStatus = "FAILED"    // 执行失败
)

// Execution 执行记录实体
// 代表一次订单的执行结果
type Execution struct {
	// 执行 ID，全局唯一标识
	ExecutionID string
	// 订单 ID，关联的订单
	OrderID string
	// 用户 ID，发起订单的用户
	UserID string
	// 交易对符号，例如 "BTC/USD"
	Symbol string
	// 买卖方向，"buy" 或 "sell"
	Side string
	// 执行价格，成交的单价
	ExecutedPrice decimal.Decimal
	// 执行数量，成交的数量
	ExecutedQuantity decimal.Decimal
	// 执行状态
	Status ExecutionStatus
	// 创建时间
	CreatedAt time.Time
	// 更新时间
	UpdatedAt time.Time
}

// ExecutionRepository 执行记录仓储接口
// 定义执行记录的持久化操作
type ExecutionRepository interface {
	// Save 保存执行记录
	// 将执行领域对象持久化到存储介质
	Save(ctx context.Context, execution *Execution) error

	// Get 获取执行记录
	// 根据执行 ID 查询执行记录
	Get(ctx context.Context, executionID string) (*Execution, error)

	// GetByOrder 获取订单执行历史
	// 查询指定订单的所有执行记录
	GetByOrder(ctx context.Context, orderID string) ([]*Execution, error)

	// GetByUser 获取用户执行历史
	// 分页查询指定用户的执行记录
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]*Execution, int64, error)
}
