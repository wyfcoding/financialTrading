package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algos/types"
)

// OrderBookSnapshot 订单簿快照
type OrderBookSnapshot struct {
	ID             uint            `json:"id"`
	SnapshotID     string          `json:"snapshot_id"`
	Symbol         string          `json:"symbol"`
	Timestamp      time.Time       `json:"timestamp"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	SequenceNumber int64           `json:"sequence_number"`
	Bids           []*PriceLevel   `json:"bids"`
	Asks           []*PriceLevel   `json:"asks"`
	TotalBidVolume float64         `json:"total_bid_volume"`
	TotalAskVolume float64         `json:"total_ask_volume"`
	BestBid        float64         `json:"best_bid"`
	BestAsk        float64         `json:"best_ask"`
	Spread         float64         `json:"spread"`
	MidPrice       float64         `json:"mid_price"`
	MarketDepth    *MarketDepth    `json:"market_depth"`
	Metadata       json.RawMessage `json:"metadata"`
}

// MarketDepth 市场深度
type MarketDepth struct {
	Levels         int       `json:"levels"`
	BidDepth       []float64 `json:"bid_depth"`       // 各价格档位的买单量
	AskDepth       []float64 `json:"ask_depth"`       // 各价格档位的卖单量
	BidCumulative  []float64 `json:"bid_cumulative"`  // 买单累计量
	AskCumulative  []float64 `json:"ask_cumulative"`  // 卖单累计量
	DepthImbalance float64   `json:"depth_imbalance"` // 深度不平衡度
}

// SnapshotManager 快照管理器
type SnapshotManager struct {
	snapshotRepo SnapshotRepository
	engine       *MatchingEngine
	snapshotChan chan *OrderBookSnapshot
	mu           sync.RWMutex
	config       *SnapshotConfig
}

// SnapshotConfig 快照配置
type SnapshotConfig struct {
	SnapshotInterval   time.Duration `json:"snapshot_interval"`
	MaxSnapshots       int           `json:"max_snapshots"`
	RetentionPeriod    time.Duration `json:"retention_period"`
	CompressionEnabled bool          `json:"compression_enabled"`
	Levels             int           `json:"levels"` // 快照深度级别
	AutoCleanup        bool          `json:"auto_cleanup"`
}

// NewSnapshotManager 创建快照管理器
func NewSnapshotManager(engine *MatchingEngine, repo SnapshotRepository, config *SnapshotConfig) *SnapshotManager {
	if config == nil {
		config = &SnapshotConfig{
			SnapshotInterval:   1 * time.Second,
			MaxSnapshots:       10000,
			RetentionPeriod:    7 * 24 * time.Hour,
			CompressionEnabled: true,
			Levels:             10,
			AutoCleanup:        true,
		}
	}

	return &SnapshotManager{
		snapshotRepo: repo,
		engine:       engine,
		snapshotChan: make(chan *OrderBookSnapshot, 1000),
		config:       config,
	}
}

// Start 启动快照管理器
func (sm *SnapshotManager) Start(ctx context.Context) error {
	// 启动快照生成器
	go sm.generateSnapshots(ctx)

	// 启动快照处理器
	go sm.processSnapshots(ctx)

	// 启动清理任务
	if sm.config.AutoCleanup {
		go sm.cleanupSnapshots(ctx)
	}

	return nil
}

// generateSnapshots 生成快照
func (sm *SnapshotManager) generateSnapshots(ctx context.Context) {
	ticker := time.NewTicker(sm.config.SnapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snapshot, err := sm.createSnapshot()
			if err != nil {
				fmt.Printf("Failed to create snapshot: %v\n", err)
				continue
			}

			// 发送到处理通道
			select {
			case sm.snapshotChan <- snapshot:
				// 成功发送
			default:
				// 通道满，丢弃快照
				fmt.Printf("Snapshot channel full, dropping snapshot: %s\n", snapshot.SnapshotID)
			}
		}
	}
}

// processSnapshots 处理快照
func (sm *SnapshotManager) processSnapshots(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case snapshot := <-sm.snapshotChan:
			err := sm.saveSnapshot(ctx, snapshot)
			if err != nil {
				fmt.Printf("Failed to save snapshot: %v\n", err)
			}
		}
	}
}

// createSnapshot 创建快照
func (sm *SnapshotManager) createSnapshot() (*OrderBookSnapshot, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 获取订单簿深度
	depth, err := sm.engine.GetOrderBookDepth(sm.config.Levels)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book depth: %w", err)
	}

	// 获取市场数据
	marketData := sm.engine.GetMarketData()

	// 计算市场深度
	marketDepth := sm.calculateMarketDepth(depth)

	// 创建快照
	snapshot := &OrderBookSnapshot{
		SnapshotID:     generateSnapshotID(),
		Symbol:         sm.engine.symbol,
		Timestamp:      time.Now(),
		SequenceNumber: time.Now().UnixNano(),
		Bids:           depth.Bids,
		Asks:           depth.Asks,
		TotalBidVolume: sm.calculateTotalVolume(depth.Bids),
		TotalAskVolume: sm.calculateTotalVolume(depth.Asks),
		BestBid:        marketData.Bid,
		BestAsk:        marketData.Ask,
		Spread:         marketData.Ask - marketData.Bid,
		MidPrice:       (marketData.Bid + marketData.Ask) / 2,
		MarketDepth:    marketDepth,
	}

	return snapshot, nil
}

// calculateMarketDepth 计算市场深度
func (sm *SnapshotManager) calculateMarketDepth(depth *OrderBookDepth) *MarketDepth {
	md := &MarketDepth{
		Levels:        len(depth.Bids),
		BidDepth:      make([]float64, len(depth.Bids)),
		AskDepth:      make([]float64, len(depth.Asks)),
		BidCumulative: make([]float64, len(depth.Bids)),
		AskCumulative: make([]float64, len(depth.Asks)),
	}

	// 计算买单深度
	var bidCumulative float64
	for i, level := range depth.Bids {
		qty := level.Quantity.InexactFloat64()
		md.BidDepth[i] = qty
		bidCumulative += qty
		md.BidCumulative[i] = bidCumulative
	}

	// 计算卖单深度
	var askCumulative float64
	for i, level := range depth.Asks {
		qty := level.Quantity.InexactFloat64()
		md.AskDepth[i] = qty
		askCumulative += qty
		md.AskCumulative[i] = askCumulative
	}

	// 计算深度不平衡度
	if bidCumulative+askCumulative > 0 {
		md.DepthImbalance = (bidCumulative - askCumulative) / (bidCumulative + askCumulative)
	}

	return md
}

// calculateTotalVolume 计算总成交量
func (sm *SnapshotManager) calculateTotalVolume(levels []*PriceLevel) float64 {
	var total decimal.Decimal
	for _, level := range levels {
		total = total.Add(level.Quantity)
	}
	return total.InexactFloat64()
}

// saveSnapshot 保存快照
func (sm *SnapshotManager) saveSnapshot(ctx context.Context, snapshot *OrderBookSnapshot) error {
	// 压缩快照数据
	if sm.config.CompressionEnabled {
		snapshot = sm.compressSnapshot(snapshot)
	}

	// 保存到存储
	err := sm.snapshotRepo.SaveSnapshot(ctx, snapshot)
	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	return nil
}

// compressSnapshot 压缩快照
func (sm *SnapshotManager) compressSnapshot(snapshot *OrderBookSnapshot) *OrderBookSnapshot {
	if snapshot == nil {
		return nil
	}

	compressed := &OrderBookSnapshot{
		SnapshotID:     snapshot.SnapshotID,
		Symbol:         snapshot.Symbol,
		Timestamp:      snapshot.Timestamp,
		SequenceNumber: snapshot.SequenceNumber,
		BestBid:        snapshot.BestBid,
		BestAsk:        snapshot.BestAsk,
		Spread:         snapshot.Spread,
		MidPrice:       snapshot.MidPrice,
	}

	maxLevels := sm.config.Levels
	if maxLevels <= 0 {
		maxLevels = 10
	}
	compressed.Bids = cloneAndQuantizeLevels(snapshot.Bids, maxLevels)
	compressed.Asks = cloneAndQuantizeLevels(snapshot.Asks, maxLevels)
	compressed.TotalBidVolume = sm.calculateTotalVolume(compressed.Bids)
	compressed.TotalAskVolume = sm.calculateTotalVolume(compressed.Asks)

	if snapshot.MarketDepth != nil {
		md := *snapshot.MarketDepth
		md.Levels = minInt(maxLevels, minInt(len(md.BidDepth), len(md.AskDepth)))
		md.BidDepth = trimFloats(md.BidDepth, md.Levels)
		md.AskDepth = trimFloats(md.AskDepth, md.Levels)
		md.BidCumulative = trimFloats(md.BidCumulative, md.Levels)
		md.AskCumulative = trimFloats(md.AskCumulative, md.Levels)
		compressed.MarketDepth = &md
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"compressed":          true,
		"levels_kept":         maxLevels,
		"original_bid_levels": len(snapshot.Bids),
		"original_ask_levels": len(snapshot.Asks),
		"compressed_at":       time.Now().UnixNano(),
	})
	compressed.Metadata = meta
	return compressed
}

// cleanupSnapshots 清理过期快照
func (sm *SnapshotManager) cleanupSnapshots(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.cleanupOldSnapshots(ctx)
			sm.cleanupExcessSnapshots(ctx)
		}
	}
}

// cleanupOldSnapshots 清理旧快照
func (sm *SnapshotManager) cleanupOldSnapshots(ctx context.Context) {
	cutoffTime := time.Now().Add(-sm.config.RetentionPeriod)

	err := sm.snapshotRepo.DeleteSnapshotsBefore(ctx, sm.engine.symbol, cutoffTime)
	if err != nil {
		fmt.Printf("Failed to cleanup old snapshots: %v\n", err)
	}
}

// cleanupExcessSnapshots 清理超额快照
func (sm *SnapshotManager) cleanupExcessSnapshots(ctx context.Context) {
	count, err := sm.snapshotRepo.CountSnapshots(ctx, sm.engine.symbol)
	if err != nil {
		fmt.Printf("Failed to count snapshots: %v\n", err)
		return
	}

	if count > int64(sm.config.MaxSnapshots) {
		excess := count - int64(sm.config.MaxSnapshots)
		err = sm.snapshotRepo.DeleteOldestSnapshots(ctx, sm.engine.symbol, int(excess))
		if err != nil {
			fmt.Printf("Failed to cleanup excess snapshots: %v\n", err)
		}
	}
}

// GetSnapshot 获取快照
func (sm *SnapshotManager) GetSnapshot(ctx context.Context, snapshotID string) (*OrderBookSnapshot, error) {
	return sm.snapshotRepo.GetSnapshot(ctx, snapshotID)
}

// GetSnapshotsByTimeRange 获取时间范围内的快照
func (sm *SnapshotManager) GetSnapshotsByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*OrderBookSnapshot, error) {
	return sm.snapshotRepo.GetSnapshotsByTimeRange(ctx, sm.engine.symbol, startTime, endTime)
}

// GetLatestSnapshot 获取最新快照
func (sm *SnapshotManager) GetLatestSnapshot(ctx context.Context) (*OrderBookSnapshot, error) {
	return sm.snapshotRepo.GetLatestSnapshot(ctx, sm.engine.symbol)
}

// ReconstructOrderBook 从快照重建订单簿
// Note: This returns *MatchingEngine for simplicity or we can implement a logic to populate it
func (sm *SnapshotManager) ReconstructOrderBook(ctx context.Context, snapshotID string) (*MatchingEngine, error) {
	snapshot, err := sm.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	// 从快照重建订单簿
	// We create a new engine and populate its book
	engine, err := NewMatchingEngine(snapshot.Symbol, 1024, nil)
	if err != nil {
		return nil, err
	}

	// Populate bids
	for _, bid := range snapshot.Bids {
		// Create a dummy order to represent this level
		order := &types.Order{
			OrderID:   fmt.Sprintf("SNAPSHOT_BID_%d", time.Now().UnixNano()),
			Symbol:    snapshot.Symbol,
			Side:      types.SideBuy,
			Price:     bid.Price,
			Quantity:  bid.Quantity,
			OrderType: types.OrderTypeLimit,
		}
		// Use internal methods to add to book
		// Note: ReplayOrder is public
		engine.ReplayOrder(order)
	}

	// Populate asks
	for _, ask := range snapshot.Asks {
		order := &types.Order{
			OrderID:   fmt.Sprintf("SNAPSHOT_ASK_%d", time.Now().UnixNano()),
			Symbol:    snapshot.Symbol,
			Side:      types.SideSell,
			Price:     ask.Price,
			Quantity:  ask.Quantity,
			OrderType: types.OrderTypeLimit,
		}
		engine.ReplayOrder(order)
	}

	return engine, nil
}

// SnapshotRepository 快照仓储接口
type SnapshotRepository interface {
	SaveSnapshot(ctx context.Context, snapshot *OrderBookSnapshot) error
	GetSnapshot(ctx context.Context, snapshotID string) (*OrderBookSnapshot, error)
	GetSnapshotsByTimeRange(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*OrderBookSnapshot, error)
	GetLatestSnapshot(ctx context.Context, symbol string) (*OrderBookSnapshot, error)
	DeleteSnapshotsBefore(ctx context.Context, symbol string, cutoffTime time.Time) error
	DeleteOldestSnapshots(ctx context.Context, symbol string, count int) error
	CountSnapshots(ctx context.Context, symbol string) (int64, error)
}

// Helper functions

func generateSnapshotID() string {
	return fmt.Sprintf("SNAPSHOT_%d", time.Now().UnixNano())
}

func cloneAndQuantizeLevels(levels []*PriceLevel, limit int) []*PriceLevel {
	if limit <= 0 || limit > len(levels) {
		limit = len(levels)
	}
	out := make([]*PriceLevel, 0, limit)
	for i := 0; i < limit; i++ {
		if levels[i] == nil {
			continue
		}
		out = append(out, &PriceLevel{
			Price:    levels[i].Price.Round(8),
			Quantity: levels[i].Quantity.Round(4),
		})
	}
	return out
}

func trimFloats(in []float64, n int) []float64 {
	if n <= 0 || len(in) == 0 {
		return []float64{}
	}
	if n > len(in) {
		n = len(in)
	}
	out := make([]float64, n)
	copy(out, in[:n])
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// EnhancedMatchingEngine 增强版撮合引擎
type EnhancedMatchingEngine struct {
	*MatchingEngine
	icebergManager     *IcebergOrderManager
	hiddenOrderManager *HiddenOrderManager
	marketMakerRule    *MarketMakerPriorityRule
	performanceMetrics *PerformanceMetrics
	snapshotManager    *SnapshotManager
	hotStandbyManager  *HotStandbyManager
}

// NewEnhancedMatchingEngine 创建增强版撮合引擎
func NewEnhancedMatchingEngine(symbol string) (*EnhancedMatchingEngine, error) {
	baseEngine, err := NewMatchingEngine(symbol, 1024, nil)
	if err != nil {
		return nil, err
	}

	return &EnhancedMatchingEngine{
		MatchingEngine:     baseEngine,
		icebergManager:     NewIcebergOrderManager(),
		hiddenOrderManager: NewHiddenOrderManager(),
		marketMakerRule:    NewMarketMakerPriorityRule([]string{}, 1.2, 0.001, 100000),
		performanceMetrics: NewPerformanceMetrics(),
	}, nil
}

// SubmitIcebergOrder 提交冰山订单
func (eme *EnhancedMatchingEngine) SubmitIcebergOrder(ctx context.Context, order *types.Order, displayQty, peakSize float64, interval time.Duration) (*MatchingResult, error) {
	// 创建冰山订单
	icebergOrder := NewIcebergOrder(order, displayQty, peakSize, interval)

	// 注册冰山订单
	eme.icebergManager.RegisterOrder(icebergOrder)

	// 提交第一个切片
	sliceOrder := icebergOrder.Slice()
	if sliceOrder == nil {
		return nil, fmt.Errorf("failed to create iceberg slice")
	}

	// 提交切片订单
	return eme.SubmitOrder(sliceOrder)
}

// SubmitHiddenOrder 提交隐藏订单
func (eme *EnhancedMatchingEngine) SubmitHiddenOrder(ctx context.Context, order *types.Order, minVisible, maxVisible float64, strategy string) (*MatchingResult, error) {
	// 创建隐藏订单
	hiddenOrder := NewHiddenOrder(order, minVisible, maxVisible, strategy)

	// 注册隐藏订单
	eme.hiddenOrderManager.RegisterOrder(hiddenOrder)

	// 获取可见数量
	visibleQty := hiddenOrder.GetVisibleQuantity()

	// 创建可见部分订单
	visibleOrder := &types.Order{
		OrderID:   fmt.Sprintf("%s_VISIBLE", order.OrderID),
		ParentID:  order.OrderID,
		Symbol:    order.Symbol,
		Side:      order.Side,
		OrderType: order.OrderType,
		Price:     order.Price,
		Quantity:  types.NewDecimalFromFloat(visibleQty),
		Timestamp: time.Now().UnixNano(),
		Status:    types.OrderStatusNew,
	}

	// 提交可见部分订单
	return eme.SubmitOrder(visibleOrder)
}

// GetPerformanceMetrics 获取性能指标
func (eme *EnhancedMatchingEngine) GetPerformanceMetrics() *PerformanceMetrics {
	return eme.performanceMetrics
}

// TakeSnapshot 手动触发快照
func (eme *EnhancedMatchingEngine) TakeSnapshot() (*OrderBookSnapshot, error) {
	if eme.snapshotManager == nil {
		return nil, fmt.Errorf("snapshot manager not initialized")
	}

	return eme.snapshotManager.createSnapshot()
}

// IcebergOrderManager 冰山订单管理器
type IcebergOrderManager struct {
	orders map[string]*IcebergOrder
	mu     sync.RWMutex
}

// NewIcebergOrderManager 创建冰山订单管理器
func NewIcebergOrderManager() *IcebergOrderManager {
	return &IcebergOrderManager{
		orders: make(map[string]*IcebergOrder),
	}
}

// RegisterOrder 注册冰山订单
func (iom *IcebergOrderManager) RegisterOrder(order *IcebergOrder) {
	iom.mu.Lock()
	defer iom.mu.Unlock()

	iom.orders[order.OrderID] = order
}

// ProcessSlices 处理切片
func (iom *IcebergOrderManager) ProcessSlices() []*types.Order {
	iom.mu.Lock()
	defer iom.mu.Unlock()

	var slices []*types.Order

	for _, order := range iom.orders {
		if slice := order.Slice(); slice != nil {
			slices = append(slices, slice)

			// 如果订单已完全切片，移除
			if order.GetRemainingQty() <= 0 {
				delete(iom.orders, order.OrderID)
			}
		}
	}

	return slices
}

// HiddenOrderManager 隐藏订单管理器
type HiddenOrderManager struct {
	orders map[string]*HiddenOrder
	mu     sync.RWMutex
}

// NewHiddenOrderManager 创建隐藏订单管理器
func NewHiddenOrderManager() *HiddenOrderManager {
	return &HiddenOrderManager{
		orders: make(map[string]*HiddenOrder),
	}
}

// RegisterOrder 注册隐藏订单
func (hom *HiddenOrderManager) RegisterOrder(order *HiddenOrder) {
	hom.mu.Lock()
	defer hom.mu.Unlock()

	hom.orders[order.OrderID] = order
}

// UpdateVisibleQuantity 更新可见数量
func (hom *HiddenOrderManager) UpdateVisibleQuantity(orderID string) {
	hom.mu.Lock()
	defer hom.mu.Unlock()

	if order, exists := hom.orders[orderID]; exists {
		// 更新可见数量策略
		// 可以根据市场条件调整可见数量
		_ = order.GetVisibleQuantity() // 触发更新
	}
}
