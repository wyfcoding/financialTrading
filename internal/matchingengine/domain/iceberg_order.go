package domain

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algos/types"
)

// IcebergOrder 冰山订单
type IcebergOrder struct {
	*types.Order
	DisplayQuantity float64       // 显示数量
	PeakSize        float64       // 峰值大小
	ReservedQty     float64       // 保留数量
	LastSliceTime   time.Time     // 最后切片时间
	SliceInterval   time.Duration // 切片间隔
	mu              sync.RWMutex
}

// NewIcebergOrder 创建冰山订单
func NewIcebergOrder(order *types.Order, displayQty, peakSize float64, interval time.Duration) *IcebergOrder {
	qty := order.Quantity.InexactFloat64()
	if displayQty <= 0 {
		displayQty = qty * 0.1 // 默认显示10%
	}

	if peakSize <= 0 {
		peakSize = displayQty * 2 // 默认峰值是显示量的2倍
	}

	if interval <= 0 {
		interval = 5 * time.Second // 默认5秒间隔
	}

	return &IcebergOrder{
		Order:           order,
		DisplayQuantity: displayQty,
		PeakSize:        peakSize,
		ReservedQty:     qty,
		SliceInterval:   interval,
		LastSliceTime:   time.Now(),
	}
}

// GetVisibleQuantity 获取可见数量
func (io *IcebergOrder) GetVisibleQuantity() float64 {
	io.mu.RLock()
	defer io.mu.RUnlock()

	// 如果保留数量小于显示数量，返回保留数量
	if io.ReservedQty < io.DisplayQuantity {
		return io.ReservedQty
	}

	// 检查是否达到切片时间
	if time.Since(io.LastSliceTime) >= io.SliceInterval {
		return io.DisplayQuantity
	}

	// 返回当前可见数量（不超过显示数量）
	visibleQty := io.DisplayQuantity
	if io.ReservedQty < visibleQty {
		visibleQty = io.ReservedQty
	}

	return visibleQty
}

// Slice 切片冰山订单
func (io *IcebergOrder) Slice() *types.Order {
	io.mu.Lock()
	defer io.mu.Unlock()

	// 检查是否还有保留数量
	if io.ReservedQty <= 0 {
		return nil
	}

	// 计算本次切片数量
	var sliceQty float64

	// 如果达到切片时间，释放一个切片
	if time.Since(io.LastSliceTime) >= io.SliceInterval {
		// 计算切片数量（不超过峰值大小）
		if io.ReservedQty > io.PeakSize {
			sliceQty = io.PeakSize
		} else {
			sliceQty = io.ReservedQty
		}

		// 更新最后切片时间
		io.LastSliceTime = time.Now()
	} else {
		// 未到切片时间，返回可见部分
		sliceQty = io.GetVisibleQuantity()
	}

	// 更新保留数量
	io.ReservedQty -= sliceQty

	// 创建切片订单
	sliceOrder := &types.Order{
		OrderID:     fmt.Sprintf("%s_SLICE_%d", io.OrderID, time.Now().UnixNano()),
		Symbol:      io.Symbol,
		Side:        io.Side,
		InstType:    io.InstType,
		OrderType:   types.OrderTypeLimit,
		Price:       io.Price,
		Quantity:    types.NewDecimalFromFloat(sliceQty),
		Timestamp:   time.Now().UnixNano(),
		Status:      types.OrderStatusNew,
		TimeInForce: io.TimeInForce,
		IsIceberg:   true,
		ParentID:    io.OrderID,
	}

	return sliceOrder
}

// IsSliced 是否已切片
func (io *IcebergOrder) IsSliced() bool {
	io.mu.RLock()
	defer io.mu.RUnlock()

	return io.ReservedQty < io.Quantity.InexactFloat64()
}

// GetRemainingQty 获取剩余数量
func (io *IcebergOrder) GetRemainingQty() float64 {
	io.mu.RLock()
	defer io.mu.RUnlock()

	return io.ReservedQty
}

// HiddenOrder 隐藏订单
type HiddenOrder struct {
	*types.Order
	MinVisibleQty   float64 // 最小可见数量
	MaxVisibleQty   float64 // 最大可见数量
	DisplayStrategy string  // 显示策略: FIXED, RANDOM, ADAPTIVE
	mu              sync.RWMutex
}

// NewHiddenOrder 创建隐藏订单
func NewHiddenOrder(order *types.Order, minVisible, maxVisible float64, strategy string) *HiddenOrder {
	if minVisible <= 0 {
		minVisible = 0.01 // 最小可见0.01%
	}

	if maxVisible <= 0 {
		maxVisible = 0.1 // 最大可见10%
	}

	if strategy == "" {
		strategy = "ADAPTIVE"
	}

	return &HiddenOrder{
		Order:           order,
		MinVisibleQty:   minVisible,
		MaxVisibleQty:   maxVisible,
		DisplayStrategy: strategy,
	}
}

// GetVisibleQuantity 获取可见数量
func (ho *HiddenOrder) GetVisibleQuantity() float64 {
	ho.mu.RLock()
	defer ho.mu.RUnlock()

	qty := ho.Quantity.InexactFloat64()

	switch ho.DisplayStrategy {
	case "FIXED":
		// 固定比例
		return qty * ho.MinVisibleQty

	case "RANDOM":
		// 随机比例
		ratio := ho.MinVisibleQty + (ho.MaxVisibleQty-ho.MinVisibleQty)*0.5 // 简化随机
		return qty * ratio

	case "ADAPTIVE":
		// 自适应策略
		// 基于市场深度和波动性调整
		marketDepth := ho.estimateMarketDepth()
		volatility := ho.estimateVolatility()

		// 深度越大，显示比例越小；波动性越大，显示比例越大
		ratio := ho.MinVisibleQty + (ho.MaxVisibleQty-ho.MinVisibleQty)*(volatility/(marketDepth+1))
		if ratio > ho.MaxVisibleQty {
			ratio = ho.MaxVisibleQty
		}

		return qty * ratio

	default:
		return qty * ho.MinVisibleQty
	}
}

// estimateMarketDepth 估算市场深度
func (ho *HiddenOrder) estimateMarketDepth() float64 {
	qty := ho.Quantity.InexactFloat64()
	if qty <= 0 {
		return 1
	}

	price := ho.Price.InexactFloat64()
	if price <= 0 {
		price = 1
	}

	// 以订单名义价值估算可成交深度，并给最小深度兜底。
	notional := qty * price
	depth := math.Max(notional*8, qty*100)

	if ho.OrderType == types.OrderTypeMarket {
		depth *= 0.7
	}

	if depth < 1000 {
		depth = 1000
	}

	return depth
}

// estimateVolatility 估算波动性
func (ho *HiddenOrder) estimateVolatility() float64 {
	base := 0.02 // 默认2%

	qty := ho.Quantity.InexactFloat64()
	sizeFactor := math.Min(qty/100000, 0.05)

	ageFactor := 0.0
	if ho.Timestamp > 0 {
		age := time.Since(time.Unix(0, ho.Timestamp))
		switch {
		case age < 5*time.Second:
			ageFactor = 0.02
		case age < 1*time.Minute:
			ageFactor = 0.01
		case age < 5*time.Minute:
			ageFactor = 0.005
		}
	}

	orderTypeFactor := 0.0
	if ho.OrderType == types.OrderTypeMarket {
		orderTypeFactor = 0.02
	}

	vol := base + sizeFactor + ageFactor + orderTypeFactor
	if vol < 0.005 {
		return 0.005
	}
	if vol > 0.2 {
		return 0.2
	}
	return vol
}

// MarketMakerPriorityRule 做市商优先撮合规则
type MarketMakerPriorityRule struct {
	MarketMakerIDs map[string]bool // 做市商ID列表
	PriorityFactor float64         // 优先因子 (1.0-2.0)
	MinSpread      float64         // 最小价差要求
	MaxOrderSize   float64         // 最大订单规模
}

// NewMarketMakerPriorityRule 创建做市商优先规则
func NewMarketMakerPriorityRule(marketMakerIDs []string, priorityFactor, minSpread, maxOrderSize float64) *MarketMakerPriorityRule {
	mmIDs := make(map[string]bool)
	for _, id := range marketMakerIDs {
		mmIDs[id] = true
	}

	if priorityFactor < 1.0 {
		priorityFactor = 1.0
	}
	if priorityFactor > 2.0 {
		priorityFactor = 2.0
	}

	return &MarketMakerPriorityRule{
		MarketMakerIDs: mmIDs,
		PriorityFactor: priorityFactor,
		MinSpread:      minSpread,
		MaxOrderSize:   maxOrderSize,
	}
}

// IsMarketMaker 检查是否为做市商
func (mm *MarketMakerPriorityRule) IsMarketMaker(userID string) bool {
	return mm.MarketMakerIDs[userID]
}

// ApplyPriority 应用优先规则
func (mm *MarketMakerPriorityRule) ApplyPriority(order *types.Order, spread float64) bool {
	// 检查是否为做市商订单
	if !mm.IsMarketMaker(order.UserID) {
		return false
	}

	// 检查价差要求
	if spread < mm.MinSpread {
		return false
	}

	// 检查订单规模
	if order.Quantity.InexactFloat64() > mm.MaxOrderSize {
		return false
	}

	return true
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	Timestamp   time.Time
	TotalOrders int64
	TotalTrades int64
	TotalVolume float64
	TotalValue  float64
	AvgLatency  time.Duration
	P50Latency  time.Duration
	P90Latency  time.Duration
	P99Latency  time.Duration
	MaxLatency  time.Duration
	Throughput  float64 // 每秒订单数
	QueueSize   int
	ErrorRate   float64
	MemoryUsage float64
	CPUUsage    float64
	mu          sync.RWMutex
}

// NewPerformanceMetrics 创建性能指标
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		Timestamp: time.Now(),
	}
}

// RecordOrder 记录订单处理
func (pm *PerformanceMetrics) RecordOrder(latency time.Duration, success bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.TotalOrders++

	// 更新延迟统计
	pm.updateLatencyStats(latency)

	if !success {
		pm.ErrorRate = float64(pm.TotalOrders) / float64(pm.TotalOrders+1)
	}

	// 更新吞吐量
	pm.updateThroughput()
}

// RecordTrade 记录交易
func (pm *PerformanceMetrics) RecordTrade(volume, value float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.TotalTrades++
	pm.TotalVolume += volume
	pm.TotalValue += value
}

// updateLatencyStats 更新延迟统计
func (pm *PerformanceMetrics) updateLatencyStats(latency time.Duration) {
	// 简化实现，实际需要维护延迟分布
	pm.AvgLatency = (pm.AvgLatency*time.Duration(pm.TotalOrders-1) + latency) / time.Duration(pm.TotalOrders)

	if latency > pm.MaxLatency {
		pm.MaxLatency = latency
	}

	// 百分位计算需要历史数据，这里简化处理
	pm.P50Latency = pm.AvgLatency
	pm.P90Latency = pm.AvgLatency * 2
	pm.P99Latency = pm.AvgLatency * 3
}

// updateThroughput 更新吞吐量
func (pm *PerformanceMetrics) updateThroughput() {
	elapsed := time.Since(pm.Timestamp).Seconds()
	if elapsed > 0 {
		pm.Throughput = float64(pm.TotalOrders) / elapsed
	}
}

// HotStandbyManager 热备管理器
type HotStandbyManager struct {
	PrimaryEngine   *MatchingEngine
	StandbyEngine   *MatchingEngine
	HealthChecker   *HealthChecker
	FailoverTrigger *FailoverTrigger
	StateManager    *StateManager
	mu              sync.RWMutex
}

// NewHotStandbyManager 创建热备管理器
func NewHotStandbyManager(primary, standby *MatchingEngine) *HotStandbyManager {
	return &HotStandbyManager{
		PrimaryEngine:   primary,
		StandbyEngine:   standby,
		HealthChecker:   NewHealthChecker(),
		FailoverTrigger: NewFailoverTrigger(),
		StateManager:    NewStateManager(),
	}
}

// Start 启动热备管理
func (hsm *HotStandbyManager) Start(ctx context.Context) error {
	// 启动健康检查
	go hsm.HealthChecker.Monitor(hsm.PrimaryEngine, hsm.StandbyEngine)

	// 启动状态同步
	go hsm.StateManager.SyncState(hsm.PrimaryEngine, hsm.StandbyEngine)

	// 监听故障转移事件
	go hsm.monitorFailover(ctx)

	return nil
}

// monitorFailover 监控故障转移
func (hsm *HotStandbyManager) monitorFailover(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-hsm.HealthChecker.HealthEvents:
			if event.Type == "PRIMARY_FAILURE" {
				hsm.triggerFailover()
			}
		}
	}
}

// triggerFailover 触发故障转移
func (hsm *HotStandbyManager) triggerFailover() {
	hsm.mu.Lock()
	defer hsm.mu.Unlock()

	// 切换主备
	hsm.PrimaryEngine, hsm.StandbyEngine = hsm.StandbyEngine, hsm.PrimaryEngine

	// 记录故障转移事件
	hsm.FailoverTrigger.RecordFailover()
}

// HealthChecker 健康检查器
type HealthChecker struct {
	HealthEvents chan *HealthEvent
	mu           sync.RWMutex
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		HealthEvents: make(chan *HealthEvent, 100),
	}
}

// Monitor 监控引擎健康状态
func (hc *HealthChecker) Monitor(primary, standby *MatchingEngine) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 检查主引擎健康状态
		if !hc.checkEngineHealth(primary) {
			hc.HealthEvents <- &HealthEvent{
				Type:      "PRIMARY_FAILURE",
				Timestamp: time.Now(),
				Details:   "Primary engine health check failed",
			}
		}

		// 检查备引擎健康状态
		if !hc.checkEngineHealth(standby) {
			hc.HealthEvents <- &HealthEvent{
				Type:      "STANDBY_FAILURE",
				Timestamp: time.Now(),
				Details:   "Standby engine health check failed",
			}
		}
	}
}

// checkEngineHealth 检查引擎健康状态
func (hc *HealthChecker) checkEngineHealth(engine *MatchingEngine) bool {
	if engine == nil || engine.orderBook == nil {
		return false
	}
	defer func() {
		_ = recover()
	}()

	md := engine.GetMarketData()
	if md.Bid > 0 && md.Ask > 0 && md.Bid > md.Ask {
		return false
	}
	if len(engine.orderBook.OrderIndex) < 0 {
		return false
	}
	if engine.GetStatus() == StatusClosed {
		return false
	}
	return true
}

// HealthEvent 健康事件
type HealthEvent struct {
	Type      string
	Timestamp time.Time
	Details   string
}

// FailoverTrigger 故障转移触发器
type FailoverTrigger struct {
	FailoverCount int64
	LastFailover  time.Time
	mu            sync.RWMutex
}

// NewFailoverTrigger 创建故障转移触发器
func NewFailoverTrigger() *FailoverTrigger {
	return &FailoverTrigger{}
}

// RecordFailover 记录故障转移
func (ft *FailoverTrigger) RecordFailover() {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	ft.FailoverCount++
	ft.LastFailover = time.Now()
}

// StateManager 状态管理器
type StateManager struct {
	SyncInterval time.Duration
	mu           sync.RWMutex
}

// NewStateManager 创建状态管理器
func NewStateManager() *StateManager {
	return &StateManager{
		SyncInterval: 1 * time.Second,
	}
}

// SyncState 同步状态
func (sm *StateManager) SyncState(primary, standby *MatchingEngine) {
	ticker := time.NewTicker(sm.SyncInterval)
	defer ticker.Stop()

	for range ticker.C {
		// 同步订单簿状态
		sm.syncOrderBook(primary, standby)

		// 同步交易历史
		sm.syncTradeHistory(primary, standby)

		// 同步市场数据
		sm.syncMarketData(primary, standby)
	}
}

// syncOrderBook 同步订单簿
func (sm *StateManager) syncOrderBook(primary, standby *MatchingEngine) {
	if primary == nil || standby == nil {
		return
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	snapshot := primary.GetEngineOrderBookSnapshot(50)
	if snapshot == nil {
		return
	}

	newBook := NewEngineOrderBook(snapshot.Symbol)
	now := time.Now().UnixNano()

	for i, level := range snapshot.Bids {
		if level == nil || level.Quantity.IsZero() {
			continue
		}
		order := &types.Order{
			OrderID:     fmt.Sprintf("SYNC_BID_%d_%d", now, i),
			Symbol:      snapshot.Symbol,
			Side:        types.SideBuy,
			OrderType:   types.OrderTypeLimit,
			Price:       level.Price,
			Quantity:    level.Quantity,
			Timestamp:   time.Now().UnixNano(),
			Status:      types.OrderStatusNew,
			TimeInForce: types.TIFGTC,
		}
		standby.addToOrderBook(order, newBook.Bids, -level.Price.InexactFloat64())
	}

	for i, level := range snapshot.Asks {
		if level == nil || level.Quantity.IsZero() {
			continue
		}
		order := &types.Order{
			OrderID:     fmt.Sprintf("SYNC_ASK_%d_%d", now, i),
			Symbol:      snapshot.Symbol,
			Side:        types.SideSell,
			OrderType:   types.OrderTypeLimit,
			Price:       level.Price,
			Quantity:    level.Quantity,
			Timestamp:   time.Now().UnixNano(),
			Status:      types.OrderStatusNew,
			TimeInForce: types.TIFGTC,
		}
		standby.addToOrderBook(order, newBook.Asks, level.Price.InexactFloat64())
	}

	standby.orderBook = newBook
}

// syncTradeHistory 同步交易历史
func (sm *StateManager) syncTradeHistory(primary, standby *MatchingEngine) {
	if primary == nil || standby == nil {
		return
	}
	if v := primary.lastPrice.Load(); v != nil {
		if price, ok := v.(decimal.Decimal); ok {
			standby.lastPrice.Store(price)
		}
	}
}

// syncMarketData 同步市场数据
func (sm *StateManager) syncMarketData(primary, standby *MatchingEngine) {
	if primary == nil || standby == nil {
		return
	}
	atomic.StoreInt32(&standby.halted, atomic.LoadInt32(&primary.halted))
	atomic.StoreInt32(&standby.status, atomic.LoadInt32(&primary.status))
}
