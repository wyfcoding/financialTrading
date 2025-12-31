// 包 撮合引擎服务的用例逻辑
package application

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
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
	engine        *algorithm.MatchingEngine                  // 撮合引擎核心
	tradeRepo     domain.TradeRepository                     // 成交记录仓储接口
	orderBookRepo domain.OrderBookRepository                 // 订单簿仓储接口
	ringBuffer    *algorithm.MpscRingBuffer[algorithm.Order] // 顶级高性能 MPSC 环形缓冲区
	logger        *slog.Logger
	stopChan      chan struct{} // 优雅退出信号
}

// NewMatchingApplicationService 创建撮合应用服务并启动后台 Sequencer
func NewMatchingApplicationService(tradeRepo domain.TradeRepository, orderBookRepo domain.OrderBookRepository, logger *slog.Logger) *MatchingApplicationService {
	rb, _ := algorithm.NewMpscRingBuffer[algorithm.Order](1048576) // 100万容量

	mas := &MatchingApplicationService{
		engine:        algorithm.NewMatchingEngine(),
		tradeRepo:     tradeRepo,
		orderBookRepo: orderBookRepo,
		ringBuffer:    rb,
		logger:        logger.With("module", "matching_engine"),
		stopChan:      make(chan struct{}),
	}

	// 启动核心 Sequencer (单线程处理器)
	// 将该协程锁定到特定的操作系统线程上，以获得更稳定的性能（Mechanical Sympathy）
	go func() {
		runtime.LockOSThread()
		mas.startSequencer()
	}()

	return mas
}

// Close 优雅关闭服务
func (mas *MatchingApplicationService) Close() {
	close(mas.stopChan)
}

// startSequencer 是整个撮合系统的核心逻辑循环
func (mas *MatchingApplicationService) startSequencer() {
	for {
		select {
		case <-mas.stopChan:
			mas.logger.Info("sequencer stopping...")
			return
		default:
			// 从 RingBuffer 轮询订单 (无锁且缓存友好)
			order := mas.ringBuffer.Poll()
			if order == nil {
				runtime.Gosched() // 队列为空，让出时间片
				continue
			}

			// 执行撮合逻辑
			// 由于是单线程访问 engine，内部完全不需要互斥锁
			trades := mas.engine.Match(order)

			// 持久化成交记录 (顶级架构：这里可以改为批量异步写入以提高 IOPS)
			if len(trades) > 0 {
				mas.asyncPersistTrades(trades)
			}

			// 直接通过订单携带的通道返回结果 (零全局锁)
			if order.ResultChan != nil {
				order.ResultChan <- &domain.MatchingResult{
					OrderID:           order.OrderID,
					Trades:            trades,
					RemainingQuantity: order.Quantity,
					Status:            "PROCESSED",
				}
			}
		}
	}
}

// asyncPersistTrades 异步持久化成交记录
func (mas *MatchingApplicationService) asyncPersistTrades(trades []*algorithm.Trade) {
	// 在生产环境中，这里通常会发送到另一个专用的持久化环形队列
	// 或者批量存入缓存/数据库
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, t := range trades {
			if err := mas.tradeRepo.Save(ctx, t); err != nil {
				mas.logger.Error("failed to persist trade", "trade_id", t.TradeID, "error", err)
			}
		}
	}()
}

// SubmitOrder 提交订单进行撮合
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

	// 1. 获取并初始化订单对象
	order := algorithm.AcquireOrder()
	order.OrderID = req.OrderID
	order.Symbol = req.Symbol
	order.Side = req.Side
	order.Price = price
	order.Quantity = quantity
	order.Timestamp = time.Now().UnixNano()

	// 2. 准备本地结果通道 (避免全局锁)
	resultChan := make(chan any, 1)
	order.ResultChan = resultChan

	// 3. 压入 RingBuffer
	if !mas.ringBuffer.Offer(order) {
		algorithm.ReleaseOrder(order)
		return nil, fmt.Errorf("matching engine is overloaded")
	}

	// 4. 等待同步结果 (或支持完全异步下单)
	select {
	case res := <-resultChan:
		algorithm.ReleaseOrder(order) // 回收对象
		return res.(*domain.MatchingResult), nil
	case <-ctx.Done():
		return nil, ctx.Err()
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
