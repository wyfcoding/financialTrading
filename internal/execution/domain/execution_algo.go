package domain

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/eventsourcing"
)

type TradeSide string

const (
	TradeSideBuy  TradeSide = "BUY"
	TradeSideSell TradeSide = "SELL"
)

type AlgoType string

const (
	AlgoTypeTWAP AlgoType = "TWAP"
	AlgoTypeVWAP AlgoType = "VWAP"
	AlgoTypePOV  AlgoType = "POV"
	AlgoTypeSOR  AlgoType = "SOR"
)

type Venue struct {
	ID           string
	Name         string
	ExecutionFee decimal.Decimal
	Latency      time.Duration
	Weight       float64
}

type VenueDepth struct {
	VenueID string
	Symbol  string
	Asks    []PriceLevel
	Bids    []PriceLevel
}

type PriceLevel struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

type AlgoOrder struct {
	eventsourcing.AggregateRoot
	ID                uint            `json:"id"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	AlgoID            string          `json:"algo_id"`
	UserID            string          `json:"user_id"`
	Symbol            string          `json:"symbol"`
	Side              TradeSide       `json:"side"`
	TotalQuantity     decimal.Decimal `json:"total_quantity"`
	ExecutedQuantity  decimal.Decimal `json:"executed_quantity"`
	ParticipationRate decimal.Decimal `json:"participation_rate"`
	AlgoType          AlgoType        `json:"algo_type"`
	StartTime         time.Time       `json:"start_time"`
	EndTime           time.Time       `json:"end_time"`
	Status            string          `json:"status"`
	StrategyParams    string          `json:"strategy_params"`
}

func NewAlgoOrder(id, userID, symbol string, side TradeSide, totalQty decimal.Decimal, algoType AlgoType, start, end time.Time, params string) *AlgoOrder {
	a := &AlgoOrder{
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
	a.SetID(id)

	a.ApplyChange(&AlgoOrderStartedEvent{
		AlgoID:    id,
		UserID:    userID,
		Symbol:    symbol,
		AlgoType:  string(algoType),
		TotalQty:  totalQty.String(),
		StartTime: start.Unix(),
	})
	return a
}

func (a *AlgoOrder) Apply(event eventsourcing.DomainEvent) {
	switch e := event.(type) {
	case *AlgoOrderStartedEvent:
		a.AlgoID = e.AlgoID
		a.UserID = e.UserID
		a.Symbol = e.Symbol
		a.AlgoType = AlgoType(e.AlgoType)
		a.TotalQuantity, _ = decimal.NewFromString(e.TotalQty)
		a.Status = "RUNNING"
	case *TradeExecutedEvent:
		executedQty, _ := decimal.NewFromString(e.Quantity)
		a.ExecutedQuantity = a.ExecutedQuantity.Add(executedQty)
		if a.ExecutedQuantity.GreaterThanOrEqual(a.TotalQuantity) {
			a.Status = "COMPLETED"
		}
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
	now := time.Now()
	if now.After(order.EndTime) {
		return nil, errors.New("order execution window passed")
	}

	profile, err := s.VolumeProfileProvider.GetProfile(order.Symbol)
	if err != nil {
		return nil, err
	}

	remainingQty := order.TotalQuantity.Sub(order.ExecutedQuantity)
	if remainingQty.IsZero() {
		return nil, nil
	}

	// 找到当前时间对应的 Slot
	currentTimeStr := now.Format("15:04")
	var currentRatio decimal.Decimal
	for i, item := range profile {
		if currentTimeStr >= item.TimeSlot {
			if i == len(profile)-1 || currentTimeStr < profile[i+1].TimeSlot {
				currentRatio = item.Ratio
				break
			}
		}
	}

	if currentRatio.IsZero() {
		return nil, nil // 不在交易时段或无成交量预测
	}

	// 按照比例计算本周期应成交量 (简化: 直接按 Ratio * TotalQty，实际应考虑已成交偏差)
	targetQty := order.TotalQuantity.Mul(currentRatio)
	if targetQty.GreaterThan(remainingQty) {
		targetQty = remainingQty
	}

	return []*ChildOrder{{
		AlgoID:    order.AlgoID,
		Symbol:    order.Symbol,
		Side:      order.Side,
		Quantity:  targetQty,
		OrderType: "MARKET", // VWAP 通常使用市价或激进限价以保证成交量匹配
		Timestamp: now.Unix(),
	}}, nil
}

// POVStrategy 百分比音量策略 (Percentage of Volume)
// 在市场成交量中保持固定的参与比例
type POVStrategy struct {
	Provider MarketDataProvider
}

func (s *POVStrategy) GenerateSlices(order *AlgoOrder) ([]*ChildOrder, error) {
	now := time.Now()
	if now.After(order.EndTime) {
		return nil, errors.New("order execution window passed")
	}

	remainingQty := order.TotalQuantity.Sub(order.ExecutedQuantity)
	if remainingQty.IsZero() {
		return nil, nil
	}

	// 获取最近一段时间的市场成交量 (e.g., 最近1分钟)
	marketVolume, err := s.Provider.GetRecentVolume(context.Background(), order.Symbol, 1*time.Minute)
	if err != nil {
		return nil, err
	}

	// 计算目标成交量: Target = MarketVolume * ParticipationRate
	// 注意：ParticipationRate 应该在 0-1 之间
	targetQty := marketVolume.Mul(order.ParticipationRate)

	// 如果目标量非常小，可能需要聚合或等待
	if targetQty.IsZero() {
		return nil, nil
	}

	if targetQty.GreaterThan(remainingQty) {
		targetQty = remainingQty
	}

	return []*ChildOrder{{
		AlgoID:    order.AlgoID,
		Symbol:    order.Symbol,
		Side:      order.Side,
		Quantity:  targetQty,
		OrderType: "MARKET",
		Timestamp: now.Unix(),
	}}, nil
}

// MarketDataProvider 提供跨市场行情深度与成交信息
type MarketDataProvider interface {
	GetVenueDepths(ctx context.Context, symbol string) ([]*VenueDepth, error)
	GetRecentVolume(ctx context.Context, symbol string, duration time.Duration) (decimal.Decimal, error)
}

// SORStrategy 智能路由策略
// 核心逻辑：在多个 Venue 之间寻找最优成交路径，平衡价格、深度、费用和延迟
type SORStrategy struct {
	Provider MarketDataProvider
	Venues   []*Venue
}

func (s *SORStrategy) GenerateSlices(order *AlgoOrder) ([]*ChildOrder, error) {
	ctx := context.Background()
	depths, err := s.Provider.GetVenueDepths(ctx, order.Symbol)
	if err != nil {
		return nil, err
	}

	remainingQty := order.TotalQuantity.Sub(order.ExecutedQuantity)
	if remainingQty.IsZero() {
		return nil, nil
	}

	optimizer := NewSOROptimizer(0.0001) // 假设一个默认的延迟因子
	routes, err := optimizer.Optimize(ctx, order.Side, remainingQty, s.Venues, depths)
	if err != nil {
		return nil, err
	}

	var slices []*ChildOrder
	now := time.Now().Unix()
	for _, r := range routes {
		slices = append(slices, &ChildOrder{
			AlgoID:    order.AlgoID,
			Symbol:    order.Symbol,
			Side:      order.Side,
			Quantity:  r.Quantity,
			Price:     r.Price,
			OrderType: "LIMIT",
			Timestamp: now,
		})
	}

	return slices, nil
}
