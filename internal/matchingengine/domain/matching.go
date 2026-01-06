// Package domain 撮合引擎的领域模型
package domain

import (
	"container/list"
	"fmt"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algorithm"
	"gorm.io/gorm"
)

// OrderLevel 表示同一价格档位下的订单集合，保证时间优先 (FIFO)
type OrderLevel struct {
	Price  decimal.Decimal
	Orders *list.List // 存储 *algorithm.Order
}

func NewOrderLevel(price decimal.Decimal) *OrderLevel {
	return &OrderLevel{
		Price:  price,
		Orders: list.New(),
	}
}

// OrderBook 内存订单簿实现 (当前版本已移除互斥锁，由单线程 Worker 独占访问)
type OrderBook struct {
	Symbol string

	// Bids 买盘：Key 为 -Price (降序)，Value 为 OrderLevel
	Bids *algorithm.SkipList[float64, *OrderLevel]
	// Asks 卖盘：Key 为 Price (升序)，Value 为 OrderLevel
	Asks *algorithm.SkipList[float64, *OrderLevel]
}

func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		Symbol: symbol,
		Bids:   algorithm.NewSkipList[float64, *OrderLevel](),
		Asks:   algorithm.NewSkipList[float64, *OrderLevel](),
	}
}

// ----------------------------------------------------------------------------
// Disruptor 模式实现：MatchingEngine (无锁核心)
// ----------------------------------------------------------------------------

// MatchTask 定义了定序队列中的任务单元
type MatchTask struct {
	Order      *algorithm.Order
	ResultChan chan *MatchingResult // 同步结果返回
}

// DisruptionEngine 是一个基于 LMAX Disruptor 思想设计的高性能撮合引擎实现。
// 核心架构：采用单线程 Worker 独占模式访问内存订单簿，完全消除锁竞争，实现极低延迟。
type DisruptionEngine struct {
	symbol    string                               // 交易对标识 (如 BTC/USDT)
	orderBook *OrderBook                           // 内存订单簿（跳表实现）
	ring      *algorithm.MpscRingBuffer[MatchTask] // 定序任务队列 (Multiple Producer Single Consumer)
	stopChan  chan struct{}                        // 停机信号
	logger    *slog.Logger                         // 结构化日志记录器
	halted    int32                                // 引擎熔断状态标识 (0: 正常, 1: 熔断)
}

func NewDisruptionEngine(symbol string, capacity uint64, logger *slog.Logger) (*DisruptionEngine, error) {
	if logger == nil {
		logger = slog.Default().With("module", "disruption_engine", "symbol", symbol)
	}
	ring, err := algorithm.NewMpscRingBuffer[MatchTask](capacity)
	if err != nil {
		return nil, fmt.Errorf("failed to create ring buffer: %w", err)
	}
	e := &DisruptionEngine{
		symbol:    symbol,
		orderBook: NewOrderBook(symbol),
		ring:      ring,
		stopChan:  make(chan struct{}),
		logger:    logger,
		halted:    0,
	}
	// 启动核心撮合 Worker (单线程)
	go e.run()
	return e, nil
}

// Halt 立即且不可逆地停止引擎运行。
// 触发场景：发生数据库写入失败等足以导致内存与持久化不一致的严重错误时。
func (e *DisruptionEngine) Halt() {
	if atomic.CompareAndSwapInt32(&e.halted, 0, 1) {
		e.logger.Error("engine halted! critical system failure detected. manual intervention required.")
	}
}

// IsHalted 检查引擎是否处于停止状态
func (e *DisruptionEngine) IsHalted() bool {
	return atomic.LoadInt32(&e.halted) == 1
}

func (e *DisruptionEngine) Symbol() string {
	return e.symbol
}

// run 是撮合引擎的核心事件循环，在独立线程中执行。
func (e *DisruptionEngine) run() {
	e.logger.Info("starting core matching worker", "symbol", e.symbol)
	// 将当前协程绑定至固定的操作系统线程，避免上下文切换抖动
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for {
		select {
		case <-e.stopChan:
			e.logger.Info("stopping core matching worker", "symbol", e.symbol)
			return
		default:
			// 检查引擎是否因严重错误已停止
			if e.IsHalted() {
				e.logger.Error("worker inactive: engine is halted")
				time.Sleep(1 * time.Second) // 避免空转导致的 CPU 占用
				continue
			}

			// 从无锁环形缓冲区轮询任务
			task := e.ring.Poll()
			if task == nil {
				runtime.Gosched()
				continue
			}

			// 执行核心撮合 (线程安全)
			result := e.applyOrder(task.Order)
			task.ResultChan <- result
		}
	}
}

// SubmitOrder 接收外部提交的订单请求并压入定序队列。
func (e *DisruptionEngine) SubmitOrder(order *algorithm.Order) (*MatchingResult, error) {
	// 入口拦截：若引擎已熔断，直接拒绝请求
	if e.IsHalted() {
		return nil, fmt.Errorf("engine is halted due to a critical persistence error")
	}

	resChan := make(chan *MatchingResult, 1)
	task := &MatchTask{
		Order:      order,
		ResultChan: resChan,
	}

	// 压入 RingBuffer，若队列满则立即报错（不阻塞业务线程）
	if !e.ring.Offer(task) {
		e.logger.Warn("failed to submit order: ring buffer full", "order_id", order.OrderID)
		return nil, fmt.Errorf("engine busy, ring buffer full")
	}

	return <-resChan, nil
}

// ReplayOrder 回放订单逻辑：仅用于系统启动恢复阶段，不触发新成交逻辑，仅重建订单簿。
func (e *DisruptionEngine) ReplayOrder(order *algorithm.Order) {
	// 在恢复模式下，由于是单线程初始化，无需通过 RingBuffer，直接操作订单簿。
	ob := e.orderBook
	if order.Side == "BUY" {
		e.addToOrderBook(order, ob.Bids, -order.Price.InexactFloat64())
	} else {
		e.addToOrderBook(order, ob.Asks, order.Price.InexactFloat64())
	}
	e.logger.Debug("order replayed into memory book", "order_id", order.OrderID, "rem_qty", order.Quantity.String())
}

// applyOrder 核心内部撮合入口。
func (e *DisruptionEngine) applyOrder(order *algorithm.Order) *MatchingResult {
	ob := e.orderBook
	result := &MatchingResult{
		OrderID:           order.OrderID,
		RemainingQuantity: order.Quantity,
		Status:            "PENDING",
	}

	e.logger.Debug("applying order", "order_id", order.OrderID, "side", order.Side, "price", order.Price.String(), "qty", order.Quantity.String())

	// 根据买卖方向选择对应的对手盘进行深度优先遍历撮合
	if order.Side == "BUY" {
		e.matchOrder(order, ob.Asks, result)
		if result.RemainingQuantity.IsPositive() {
			e.logger.Debug("order partially matched, adding remaining to book", "order_id", order.OrderID, "rem_qty", result.RemainingQuantity.String())
			e.addToOrderBook(order, ob.Bids, -order.Price.InexactFloat64())
		}
	} else {
		e.matchOrder(order, ob.Bids, result)
		if result.RemainingQuantity.IsPositive() {
			e.logger.Debug("order partially matched, adding remaining to book", "order_id", order.OrderID, "rem_qty", result.RemainingQuantity.String())
			e.addToOrderBook(order, ob.Asks, order.Price.InexactFloat64())
		}
	}

	if len(result.Trades) > 0 {
		if result.RemainingQuantity.IsZero() {
			result.Status = "MATCHED"
		} else {
			result.Status = "PARTIALLY_MATCHED"
		}
		e.logger.Info("order execution finished", "order_id", order.OrderID, "status", result.Status, "trades_count", len(result.Trades))
	}

	return result
}

func (e *DisruptionEngine) matchOrder(order *algorithm.Order, opponentBook *algorithm.SkipList[float64, *OrderLevel], result *MatchingResult) {
	it := opponentBook.Iterator()
	for {
		oppPriceKey, oppLevel, ok := it.Next()
		if !ok {
			break
		}

		realOppPrice := oppLevel.Price
		if order.Side == "BUY" {
			if order.Price.LessThan(realOppPrice) {
				break
			}
		} else {
			if order.Price.GreaterThan(realOppPrice) {
				break
			}
		}

		var nextOrder *list.Element
		for el := oppLevel.Orders.Front(); el != nil; el = nextOrder {
			nextOrder = el.Next()
			oppOrder := el.Value.(*algorithm.Order)

			matchQty := decimal.Min(result.RemainingQuantity, oppOrder.Quantity)

			trade := &algorithm.Trade{
				TradeID:   generateTradeID(),
				Symbol:    e.symbol,
				Price:     realOppPrice,
				Quantity:  matchQty,
				Timestamp: time.Now().UnixNano(),
			}

			if order.Side == "BUY" {
				trade.BuyOrderID = order.OrderID
				trade.SellOrderID = oppOrder.OrderID
			} else {
				trade.BuyOrderID = oppOrder.OrderID
				trade.SellOrderID = order.OrderID
			}

			result.Trades = append(result.Trades, trade)
			result.RemainingQuantity = result.RemainingQuantity.Sub(matchQty)
			oppOrder.Quantity = oppOrder.Quantity.Sub(matchQty)

			if oppOrder.Quantity.IsZero() {
				oppLevel.Orders.Remove(el)
			}
			if result.RemainingQuantity.IsZero() {
				break
			}
		}

		if oppLevel.Orders.Len() == 0 {
			opponentBook.Delete(oppPriceKey)
		}
		if result.RemainingQuantity.IsZero() {
			break
		}
	}
}

func (e *DisruptionEngine) addToOrderBook(order *algorithm.Order, book *algorithm.SkipList[float64, *OrderLevel], key float64) {
	level, ok := book.Search(key)
	if !ok {
		level = NewOrderLevel(order.Price)
		book.Insert(key, level)
	}
	orderCopy := *order
	orderCopy.Quantity = order.Quantity
	level.Orders.PushBack(&orderCopy)
}

// GetOrderBookSnapshot 获取订单簿快照
func (e *DisruptionEngine) GetOrderBookSnapshot(depth int) *OrderBookSnapshot {
	bids := e.collectLevels(e.orderBook.Bids, depth)
	asks := e.collectLevels(e.orderBook.Asks, depth)

	return &OrderBookSnapshot{
		Symbol:    e.symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now().Unix(),
	}
}

func (e *DisruptionEngine) collectLevels(book *algorithm.SkipList[float64, *OrderLevel], depth int) []*OrderBookLevel {
	levels := make([]*OrderBookLevel, 0, depth)
	it := book.Iterator()
	for i := 0; i < depth; i++ {
		_, level, ok := it.Next()
		if !ok {
			break
		}

		var totalQty decimal.Decimal
		for el := level.Orders.Front(); el != nil; el = el.Next() {
			totalQty = totalQty.Add(el.Value.(*algorithm.Order).Quantity)
		}

		levels = append(levels, &OrderBookLevel{
			Price:    level.Price,
			Quantity: totalQty,
		})
	}
	return levels
}

// generateTradeID 生成内部唯一的成交记录编号。
func generateTradeID() string {
	return "T-" + time.Now().Format("20060102150405.000000")
}

// MatchingResult 描述了订单进入引擎后的撮合最终状态。
type MatchingResult struct {
	OrderID           string             // 关联的订单 ID
	Trades            []*algorithm.Trade // 本次撮合产生的所有成交明细
	RemainingQuantity decimal.Decimal    // 订单剩余未成交的数量
	Status            string             // 最终撮合状态 (MATCHED, PARTIALLY_MATCHED 等)
}

// OrderBookLevel 描述订单簿中单一价格档位的聚合信息。
type OrderBookLevel struct {
	Price    decimal.Decimal `json:"price"`    // 该档位的委托价格
	Quantity decimal.Decimal `json:"quantity"` // 该档位所有挂单的总数量
}

// OrderBookSnapshot 存储订单簿在某一时刻的完整状态，用于持久化恢复。
type OrderBookSnapshot struct {
	gorm.Model
	Symbol    string            `gorm:"column:symbol;type:varchar(20);index;not null;comment:交易对标识"`
	Bids      []*OrderBookLevel `gorm:"-"`                                     // 内存中的买盘层级列表
	Asks      []*OrderBookLevel `gorm:"-"`                                     // 内存中的卖盘层级列表
	BidsJSON  string            `gorm:"column:bids;type:text;comment:买盘详情序列化"` // 持久化的买盘数据 (JSON)
	AsksJSON  string            `gorm:"column:asks;type:text;comment:卖盘详情序列化"` // 持久化的卖盘数据 (JSON)
	Timestamp int64             `gorm:"column:timestamp;type:bigint;comment:快照时间戳"`
}

// End of domain file
