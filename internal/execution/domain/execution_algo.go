package domain

import (
	"errors"
	"math/rand/v2"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ExecutionStrategyType 策略类型枚举
type ExecutionStrategyType string

const (
	StrategyTWAP ExecutionStrategyType = "TWAP"
	StrategyVWAP ExecutionStrategyType = "VWAP"
)

// ParentOrder 母单，需要被拆分的巨额订单
type ParentOrder struct {
	gorm.Model
	OrderID        string                `gorm:"type:varchar(64);unique_index;not null" json:"order_id"`
	Symbol         string                `gorm:"type:varchar(20);not null" json:"symbol"`
	Side           string                `gorm:"type:varchar(10);not null" json:"side"` // BUY, SELL
	TotalQuantity  decimal.Decimal       `gorm:"type:decimal(20,8);not null" json:"total_quantity"`
	ExecutedQty    decimal.Decimal       `gorm:"type:decimal(20,8);default:0" json:"executed_qty"`
	StrategyType   ExecutionStrategyType `gorm:"type:varchar(20);not null" json:"strategy_type"`
	StartTime      time.Time             `gorm:"not null" json:"start_time"`
	EndTime        time.Time             `gorm:"not null" json:"end_time"`
	Status         string                `gorm:"type:varchar(20);default:'PENDING'" json:"status"` // PENDING, ACTIVE, COMPLETED, CANCELLED
	StrategyParams string                `gorm:"type:text" json:"strategy_params"`                 // JSON string for specific params
}

// ChildOrder 子单，拆分后的实际执行单
type ChildOrder struct {
	gorm.Model
	ParentOrderID string          `gorm:"type:varchar(64);index;not null" json:"parent_order_id"`
	SliceID       int             `gorm:"not null" json:"slice_id"` // 第几片
	TargetTime    time.Time       `gorm:"not null" json:"target_time"`
	Quantity      decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"quantity"`
	PriceLimit    decimal.Decimal `gorm:"type:decimal(20,8)" json:"price_limit"` // 可选限价
	Status        string          `gorm:"type:varchar(20);default:'PENDING'" json:"status"`
}

// ExecutionStrategy 定义拆单算法接口
type ExecutionStrategy interface {
	// GenerateSlices 根据母单参数和市场状态生成剩余的子单计划
	GenerateSlices(order *ParentOrder, marketData interface{}) ([]*ChildOrder, error)
}

// TWAPStrategy 时间加权平均价格策略
// 将订单在时间上均匀切片，并加入随机扰动以隐藏踪迹
type TWAPStrategy struct {
	MinSliceSize decimal.Decimal // 最小子单数量
	Randomize    bool            // 是否添加随机噪音
}

func (s *TWAPStrategy) GenerateSlices(order *ParentOrder, _ interface{}) ([]*ChildOrder, error) {
	now := time.Now()
	if now.After(order.EndTime) {
		return nil, errors.New("order execution window passed")
	}
	startTime := order.StartTime
	if now.After(startTime) {
		startTime = now
	}

	remainingQty := order.TotalQuantity.Sub(order.ExecutedQty)
	if remainingQty.LessThanOrEqual(decimal.Zero) {
		return nil, nil
	}

	duration := order.EndTime.Sub(startTime)
	if duration <= 0 {
		return nil, errors.New("invalid time window")
	}

	// 假设每分钟切一片 (或根据最小单量决定)
	// 这里简化为固定间隔切片
	interval := 1 * time.Minute
	numSlices := int(duration / interval)
	if numSlices <= 0 {
		numSlices = 1
	}

	baseSliceQty := remainingQty.Div(decimal.NewFromInt(int64(numSlices)))

	// 如果切片太小，减少切片数量
	if baseSliceQty.LessThan(s.MinSliceSize) && s.MinSliceSize.IsPositive() {
		numSlices = int(remainingQty.Div(s.MinSliceSize).IntPart())
		if numSlices == 0 {
			numSlices = 1
		}
		baseSliceQty = remainingQty.Div(decimal.NewFromInt(int64(numSlices)))
		interval = duration / time.Duration(numSlices)
	}

	slices := make([]*ChildOrder, 0, numSlices)
	randSource := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))

	currentQtySum := decimal.Zero

	for i := 0; i < numSlices; i++ {
		qty := baseSliceQty

		// 添加随机扰动 +/- 10%
		if s.Randomize && numSlices > 1 {
			noise := (randSource.Float64() - 0.5) * 0.2 // -0.1 to 0.1
			qty = qty.Mul(decimal.NewFromFloat(1.0 + noise))
		}

		// 修正最后一片，确保总数匹配
		if i == numSlices-1 {
			qty = remainingQty.Sub(currentQtySum)
		} else {
			currentQtySum = currentQtySum.Add(qty)
		}

		// 目标执行时间也加入轻微随机延迟 (0-30s)
		delay := time.Duration(0)
		if s.Randomize {
			delay = time.Duration(randSource.Int64N(30)) * time.Second
		}

		slices = append(slices, &ChildOrder{
			ParentOrderID: order.OrderID,
			SliceID:       i + 1,
			TargetTime:    startTime.Add(time.Duration(i) * interval).Add(delay),
			Quantity:      qty,
			Status:        "PENDING",
		})
	}

	return slices, nil
}

// VolumeProfileItem 历史成交量分布数据
type VolumeProfileItem struct {
	TimeSlot string  // "09:30", "09:35" etc.
	Ratio    float64 // 该时段占全天总成交量的比例 (0.0 - 1.0)
}

// VWAPStrategy 成交量加权平均价格策略
// 根据历史成交量分布曲线 (Volume Profile) 分配子单权重
type VWAPStrategy struct {
	Profile []VolumeProfileItem
}

func (s *VWAPStrategy) GenerateSlices(order *ParentOrder, _ interface{}) ([]*ChildOrder, error) {
	// 简化实现：假设 Profile 覆盖了订单的整个执行窗口
	// 实际应用需匹配 TimeSlot 和当前时间

	remainingQty := order.TotalQuantity.Sub(order.ExecutedQty)
	if remainingQty.LessThanOrEqual(decimal.Zero) {
		return nil, nil
	}

	// 计算剩余时间窗口内的总 Profile 权重
	// 这里简化为：直接使用 Profile 的所有 bin 分配剩余数量
	// 真实逻辑需要截取 Profile 中对应 StartTime -> EndTime 的部分

	totalRatio := 0.0
	for _, item := range s.Profile {
		totalRatio += item.Ratio
	}

	slices := make([]*ChildOrder, 0, len(s.Profile))
	currentQtySum := decimal.Zero

	// 假设 Profile 的每个 Item 代表一个固定的时间窗口 (如 5 分钟)
	interval := 5 * time.Minute
	startTime := order.StartTime

	for i, item := range s.Profile {
		// 归一化比例
		normalizedRatio := item.Ratio / totalRatio
		qty := remainingQty.Mul(decimal.NewFromFloat(normalizedRatio))

		// 修正最后一片
		if i == len(s.Profile)-1 {
			qty = remainingQty.Sub(currentQtySum)
		} else {
			currentQtySum = currentQtySum.Add(qty)
		}

		slices = append(slices, &ChildOrder{
			ParentOrderID: order.OrderID,
			SliceID:       i + 1,
			TargetTime:    startTime.Add(time.Duration(i) * interval),
			Quantity:      qty,
			Status:        "PENDING",
		})
	}

	return slices, nil
}
