// 包 撮合引擎服务的用例逻辑
package application

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/pkg/algorithm"
	"github.com/wyfcoding/pkg/logging"
)

// SubmitOrderRequest 提交订单请求 DTO
type SubmitOrderRequest struct {
	OrderID  string // 订单 ID
	Symbol   string // 交易对
	Side     string // 买卖方向
	Price    string // 价格
	Quantity string // 数量
}

// MatchingApplicationService 撮合应用服务
// 负责协调撮合引擎、订单簿和成交记录的持久化
type MatchingApplicationService struct {
	engine        *algorithm.MatchingEngine              // 撮合引擎核心
	tradeRepo     domain.TradeRepository                 // 成交记录仓储接口
	orderBookRepo domain.OrderBookRepository             // 订单簿仓储接口
	queue         *algorithm.LockFreeQueue               // 高性能无锁请求队列
	resultChans   map[string]chan *domain.MatchingResult // 简单实现的结果通知机制
	mu            sync.RWMutex
}

// NewMatchingApplicationService 创建撮合应用服务并启动后台 Sequencer
func NewMatchingApplicationService(tradeRepo domain.TradeRepository, orderBookRepo domain.OrderBookRepository) *MatchingApplicationService {
	mas := &MatchingApplicationService{
		engine:        algorithm.NewMatchingEngine(),
		tradeRepo:     tradeRepo,
		orderBookRepo: orderBookRepo,
		queue:         algorithm.NewLockFreeQueue(1048576), // 100万容量的无锁队列
		resultChans:   make(map[string]chan *domain.MatchingResult),
	}

	// 启动核心 Sequencer (单线程处理器)
	go mas.startSequencer()

	return mas
}

// startSequencer 是整个撮合系统的核心。
// 它单线程地、顺序地处理所有撮合请求，彻底消除了撮合逻辑内部的锁竞争。
func (mas *MatchingApplicationService) startSequencer() {
	ctx := context.Background()
	for {
		item, ok := mas.queue.Pop()
		if !ok {
			// 队列为空时让出 CPU 或进行自旋优化
			runtime.Gosched()
			continue
		}

		order, ok := item.(*algorithm.Order)
		if !ok {
			continue
		}

		// 执行纯内存撮合（此时不需要担心 engine 内部的锁竞争，因为只有这一个线程在访问）
		trades := mas.engine.Match(order)

		// 异步持久化成交记录
		go func(ts []*algorithm.Trade, oID string) {
			for _, t := range ts {
				if err := mas.tradeRepo.Save(ctx, t); err != nil {
					logging.Error(ctx, "Failed to persist trade", "trade_id", t.TradeID, "error", err)
				}
			}
		}(trades, order.OrderID)

		// 通知结果
		mas.mu.RLock()
		ch, exists := mas.resultChans[order.OrderID]
		mas.mu.RUnlock()

		if exists {
			ch <- &domain.MatchingResult{
				OrderID:           order.OrderID,
				Trades:            trades,
				RemainingQuantity: order.Quantity,
				Status:            "MATCHED",
			}
		}
	}
}

// SubmitOrder 提交订单进行撮合 (现在的 SubmitOrder 变成了生产者)
func (mas *MatchingApplicationService) SubmitOrder(ctx context.Context, req *SubmitOrderRequest) (*domain.MatchingResult, error) {
	// 记录性能监控
	defer logging.LogDuration(ctx, "Order matching completed",
		"order_id", req.OrderID,
		"symbol", req.Symbol,
	)()

	// 验证输入
	if req.OrderID == "" || req.Symbol == "" || req.Side == "" {
		return nil, fmt.Errorf("invalid request parameters")
	}

	// 解析价格和数量
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}

	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	// 1. 使用对象池获取订单对象，显著降低 GC 压力
	order := algorithm.AcquireOrder()
	order.OrderID = req.OrderID
	order.Symbol = req.Symbol
	order.Side = req.Side
	order.Price = price
	order.Quantity = quantity
	order.Timestamp = time.Now().UnixNano()

	// 2. 注册结果通道
	resultChan := make(chan *domain.MatchingResult, 1)
	mas.mu.Lock()
	mas.resultChans[req.OrderID] = resultChan
	mas.mu.Unlock()

	defer func() {
		mas.mu.Lock()
		delete(mas.resultChans, req.OrderID)
		mas.mu.Unlock()
		close(resultChan)
	}()

	// 3. 推入无锁队列 (非阻塞操作)
	if !mas.queue.Push(order) {
		algorithm.ReleaseOrder(order)
		return nil, fmt.Errorf("matching engine queue is full")
	}

	// 4. 等待撮合结果 (设置超时)
	select {
	case result := <-resultChan:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Second):
		return nil, fmt.Errorf("matching timeout")
	}
}

// GetOrderBook 获取订单簿快照
func (mas *MatchingApplicationService) GetOrderBook(ctx context.Context, symbol string, depth int) (*domain.OrderBookSnapshot, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	if depth <= 0 {
		depth = 20
	}

	// 从引擎获取订单簿
	bids := mas.engine.GetBids(depth)
	asks := mas.engine.GetAsks(depth)

	// 转换为快照
	snapshot := &domain.OrderBookSnapshot{
		Symbol:    symbol,
		Bids:      make([]*domain.OrderBookLevel, 0, len(bids)),
		Asks:      make([]*domain.OrderBookLevel, 0, len(asks)),
		Timestamp: 0,
	}

	for _, bid := range bids {
		snapshot.Bids = append(snapshot.Bids, &domain.OrderBookLevel{
			Price:    bid.Price,
			Quantity: bid.Quantity,
		})
	}

	for _, ask := range asks {
		snapshot.Asks = append(snapshot.Asks, &domain.OrderBookLevel{
			Price:    ask.Price,
			Quantity: ask.Quantity,
		})
	}

	// 保存快照
	if err := mas.orderBookRepo.SaveSnapshot(ctx, snapshot); err != nil {
		logging.Error(ctx, "Failed to save order book snapshot",
			"symbol", symbol,
			"error", err,
		)
	}

	return snapshot, nil
}

// GetTrades 获取成交历史
func (mas *MatchingApplicationService) GetTrades(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	if limit <= 0 {
		limit = 100
	}

	trades, err := mas.tradeRepo.GetLatestTrades(ctx, symbol, limit)
	if err != nil {
		logging.Error(ctx, "Failed to get trades",
			"symbol", symbol,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}

	return trades, nil
}