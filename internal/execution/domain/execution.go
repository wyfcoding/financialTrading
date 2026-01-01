// 包 domain 执行服务的领域模型、仓储接口和领域服务。
package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ExecutionStatus 定义了订单执行的状态
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "PENDING"   // 待处理
	ExecutionStatusExecuting ExecutionStatus = "EXECUTING" // 执行中
	ExecutionStatusCompleted ExecutionStatus = "COMPLETED" // 已完成
	ExecutionStatusFailed    ExecutionStatus = "FAILED"    // 已失败
	ExecutionStatusCancelled ExecutionStatus = "CANCELLED" // 已取消
)

// OrderSide 定义了订单的买卖方向
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"  // 买单
	OrderSideSell OrderSide = "SELL" // 卖单
)

// Execution 是执行记录的领域实体
type Execution struct {
	gorm.Model
	// ExecutionID 是执行记录的唯一标识符
	ExecutionID string `gorm:"column:execution_id;type:varchar(32);uniqueIndex;not null" json:"execution_id"`
	// OrderID 是关联的订单ID
	OrderID string `gorm:"column:order_id;type:varchar(32);index;not null" json:"order_id"`
	// UserID 是发起订单的用户的ID
	UserID string `gorm:"column:user_id;type:varchar(32);index;not null" json:"user_id"`
	// Symbol 是交易对
	Symbol string `gorm:"column:symbol;type:varchar(20);not null" json:"symbol"`
	// Side 是交易方向 (BUY 或 SELL)
	Side OrderSide `gorm:"column:side;type:varchar(10);not null" json:"side"`
	// ExecutedPrice 是本次执行的成交价格
	ExecutedPrice decimal.Decimal `gorm:"column:executed_price;type:decimal(32,18);not null" json:"executed_price"`
	// ExecutedQuantity 是本次执行的成交数量
	ExecutedQuantity decimal.Decimal `gorm:"column:executed_quantity;type:decimal(32,18);not null" json:"executed_quantity"`
	// Status 是本次执行的状态
	Status ExecutionStatus `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
}

// AlgoType 定义算法类型
type AlgoType string

const (
	AlgoTypeVWAP AlgoType = "VWAP"
	AlgoTypeTWAP AlgoType = "TWAP"
)

// AlgoOrder 代表一个算法订单执行任务
type AlgoOrder struct {
	gorm.Model
	AlgoID            string          `gorm:"column:algo_id;type:varchar(36);uniqueIndex;not null" json:"algo_id"`
	UserID            string          `gorm:"column:user_id;type:varchar(32);index;not null" json:"user_id"`
	Symbol            string          `gorm:"column:symbol;type:varchar(20);not null" json:"symbol"`
	Side              OrderSide       `gorm:"column:side;type:varchar(10);not null" json:"side"`
	TotalQuantity     decimal.Decimal `gorm:"column:total_quantity;type:decimal(32,18);not null" json:"total_quantity"`
	ExecutedQuantity  decimal.Decimal `gorm:"column:executed_quantity;type:decimal(32,18);not null" json:"executed_quantity"`
	AlgoType          AlgoType        `gorm:"column:algo_type;type:varchar(20);not null" json:"algo_type"`
	StartTime         time.Time       `gorm:"column:start_time" json:"start_time"`
	EndTime           time.Time       `gorm:"column:end_time" json:"end_time"`
	ParticipationRate decimal.Decimal `gorm:"column:participation_rate;type:decimal(10,4)" json:"participation_rate"` // 0.0-1.0
	Status            ExecutionStatus `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
}

// End of domain file
