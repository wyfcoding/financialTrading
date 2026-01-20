package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TradeSide 交易方向
type TradeSide string

const (
	TradeSideBuy  TradeSide = "BUY"
	TradeSideSell TradeSide = "SELL"
)

// AlgoType 策略类型枚举
type AlgoType string

const (
	AlgoTypeTWAP AlgoType = "TWAP"
	AlgoTypeVWAP AlgoType = "VWAP"
)

// AlgoOrder 算法订单 (原 ParentOrder)
type AlgoOrder struct {
	gorm.Model
	AlgoID            string          `gorm:"type:varchar(64);unique_index;not null" json:"algo_id"`
	UserID            string          `gorm:"type:varchar(64);not null" json:"user_id"`
	Symbol            string          `gorm:"type:varchar(20);not null" json:"symbol"`
	Side              TradeSide       `gorm:"type:varchar(10);not null" json:"side"` // BUY, SELL using TradeSide
	TotalQuantity     decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"total_quantity"`
	ExecutedQuantity  decimal.Decimal `gorm:"type:decimal(20,8);default:0" json:"executed_qty"`
	ParticipationRate decimal.Decimal `gorm:"type:decimal(10,4);default:0" json:"participation_rate"` // Default 0
	AlgoType          AlgoType        `gorm:"type:varchar(20);not null" json:"algo_type"`
	StartTime         time.Time       `gorm:"not null" json:"start_time"`
	EndTime           time.Time       `gorm:"not null" json:"end_time"`
	Status            string          `gorm:"type:varchar(20);default:'PENDING'" json:"status"` // PENDING, ACTIVE, COMPLETED, CANCELLED
	StrategyParams    string          `gorm:"type:text" json:"strategy_params"`                 // JSON string for specific params
}

func NewAlgoOrder(id, userID, symbol string, side TradeSide, totalQty decimal.Decimal, algoType AlgoType, start, end time.Time, params string) *AlgoOrder {
	return &AlgoOrder{
		AlgoID:         id,
		UserID:         userID,
		Symbol:         symbol,
		Side:           side,
		TotalQuantity:  totalQty,
		AlgoType:       algoType,
		StartTime:      start,
		EndTime:        end,
		StrategyParams: params,
		Status:         "PENDING",
	}
}

// ChildOrder 子单，拆分后的实际执行单
type ChildOrder struct {
	OrderID   string          `json:"order_id"`
	AlgoID    string          `json:"algo_id"` // 关联的 AlgoOrder ID
	Symbol    string          `json:"symbol"`
	Side      TradeSide       `json:"side"`
	Quantity  decimal.Decimal `json:"quantity"`
	Price     decimal.Decimal `json:"price"` // 限价单使用，市价单可为0
	OrderType string          `json:"order_type"`
	Timestamp int64           `json:"timestamp"`
}

// ExecutionStrategy 定义拆单算法接口
type ExecutionStrategy interface {
	// GenerateSlices 根据母单参数和市场状态生成剩余的子单计划
	GenerateSlices(order *AlgoOrder) ([]*ChildOrder, error)
}

// TWAPStrategy 时间加权平均价格策略
// 将订单在时间上均匀切片，并加入随机扰动以隐藏踪迹
type TWAPStrategy struct {
	MinSliceSize decimal.Decimal // 最小子单数量
	Randomize    bool            // 是否添加随机噪音
}

func (s *TWAPStrategy) GenerateSlices(order *AlgoOrder) ([]*ChildOrder, error) {
	now := time.Now()
	if now.After(order.EndTime) {
		return nil, errors.New("order execution window passed")
	}

	remainingQty := order.TotalQuantity.Sub(order.ExecutedQuantity)
	if remainingQty.LessThanOrEqual(decimal.Zero) {
		return nil, nil // 已完成
	}

	// 计算剩余时间与切片间隔
	remainingDuration := order.EndTime.Sub(now)
	if remainingDuration <= 0 {
		// 立即执行剩余全部？或者根据策略处理超时
		// 这里简单处理：若超时但未完成，生成一个剩余量的单子（或报错）
		return []*ChildOrder{{
			AlgoID:    order.AlgoID,
			Symbol:    order.Symbol,
			Side:      order.Side,
			Quantity:  remainingQty,
			OrderType: "MARKET", // 紧急执行
			Timestamp: now.Unix(),
		}}, nil
	}

	// 简单 TWAP 逻辑：假设每分钟执行一次
	// 实际应更复杂
	// 这里仅生成下一个切片

	// 示例：仅拆分为 minSliceSize
	// 实际应根据 TotalQuantity/Duration 计算
	sliceQty := s.MinSliceSize
	if sliceQty.GreaterThan(remainingQty) {
		sliceQty = remainingQty
	}

	child := &ChildOrder{
		AlgoID:    order.AlgoID,
		Symbol:    order.Symbol,
		Side:      order.Side,
		Quantity:  sliceQty,
		OrderType: "LIMIT", // 需结合市场价格定
		Timestamp: now.Unix(),
	}

	return []*ChildOrder{child}, nil
}

// VWAPStrategy 成交量加权平均价格策略
// 根据历史成交量分布预测当日分布，按比例下单
type VWAPStrategy struct {
	VolumeProfileProvider VolumeProfileProvider
}

type VolumeProfileProvider interface {
	GetProfile(symbol string) ([]VolumeProfileItem, error)
}

type VolumeProfileItem struct {
	TimeSlot string // HH:MM
	Ratio    decimal.Decimal
}

func (s *VWAPStrategy) GenerateSlices(order *AlgoOrder) ([]*ChildOrder, error) {
	// 简化实现
	return nil, nil
}
