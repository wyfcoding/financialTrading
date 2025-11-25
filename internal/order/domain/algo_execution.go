package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// AlgoStrategy 算法策略类型
type AlgoStrategy string

const (
	AlgoStrategyTWAP AlgoStrategy = "TWAP"
	AlgoStrategyVWAP AlgoStrategy = "VWAP"
)

// AlgoOrder 算法订单
type AlgoOrder struct {
	OrderID       string
	Symbol        string
	Side          string
	TotalQuantity decimal.Decimal
	Strategy      AlgoStrategy
	StartTime     time.Time
	EndTime       time.Time
	// VWAP 特有参数：预测的成交量分布
	VolumeProfile []decimal.Decimal
}

// Slice 订单切片
type Slice struct {
	SliceID   string
	Quantity  decimal.Decimal
	ExecuteAt time.Time
}

// AlgoExecutionService 算法执行服务
type AlgoExecutionService struct{}

func NewAlgoExecutionService() *AlgoExecutionService {
	return &AlgoExecutionService{}
}

// GenerateSlices 生成订单切片
func (s *AlgoExecutionService) GenerateSlices(order *AlgoOrder) ([]Slice, error) {
	duration := order.EndTime.Sub(order.StartTime)
	if duration <= 0 {
		return nil, nil
	}

	// 假设每分钟切一次
	intervals := int(duration.Minutes())
	if intervals <= 0 {
		intervals = 1
	}

	slices := make([]Slice, 0, intervals)
	intervalDuration := duration / time.Duration(intervals)

	switch order.Strategy {
	case AlgoStrategyTWAP:
		// TWAP: 均匀切分
		sliceQty := order.TotalQuantity.Div(decimal.NewFromInt(int64(intervals)))
		for i := 0; i < intervals; i++ {
			slices = append(slices, Slice{
				SliceID:   order.OrderID + "-" + string(rune(i)), // 简化 ID
				Quantity:  sliceQty,
				ExecuteAt: order.StartTime.Add(time.Duration(i) * intervalDuration),
			})
		}
	case AlgoStrategyVWAP:
		// VWAP: 根据成交量分布切分
		// 假设 VolumeProfile 长度等于 intervals，且归一化（总和为 1）
		// 如果 VolumeProfile 为空或长度不匹配，退化为 TWAP
		if len(order.VolumeProfile) != intervals {
			// 简单处理：退化为 TWAP
			sliceQty := order.TotalQuantity.Div(decimal.NewFromInt(int64(intervals)))
			for i := 0; i < intervals; i++ {
				slices = append(slices, Slice{
					SliceID:   order.OrderID + "-" + string(rune(i)),
					Quantity:  sliceQty,
					ExecuteAt: order.StartTime.Add(time.Duration(i) * intervalDuration),
				})
			}
		} else {
			for i, ratio := range order.VolumeProfile {
				sliceQty := order.TotalQuantity.Mul(ratio)
				slices = append(slices, Slice{
					SliceID:   order.OrderID + "-" + string(rune(i)),
					Quantity:  sliceQty,
					ExecuteAt: order.StartTime.Add(time.Duration(i) * intervalDuration),
				})
			}
		}
	}

	return slices, nil
}

// CalculateVWAP 计算成交均价
func (s *AlgoExecutionService) CalculateVWAP(trades []struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
}) decimal.Decimal {
	totalPV := decimal.Zero
	totalVolume := decimal.Zero

	for _, trade := range trades {
		totalPV = totalPV.Add(trade.Price.Mul(trade.Quantity))
		totalVolume = totalVolume.Add(trade.Quantity)
	}

	if totalVolume.IsZero() {
		return decimal.Zero
	}

	return totalPV.Div(totalVolume)
}

// CalculateTWAP 计算时间加权均价
func (s *AlgoExecutionService) CalculateTWAP(prices []decimal.Decimal) decimal.Decimal {
	if len(prices) == 0 {
		return decimal.Zero
	}

	totalPrice := decimal.Zero
	for _, p := range prices {
		totalPrice = totalPrice.Add(p)
	}

	return totalPrice.Div(decimal.NewFromInt(int64(len(prices))))
}
