package domain

import (
	"container/list"
	"fmt"
	"log/slog"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	algorithm "github.com/wyfcoding/pkg/algos/structures"
	"github.com/wyfcoding/pkg/algos/types"

	"github.com/shopspring/decimal"
)

type MarketStatus int32

const (
	StatusInit    MarketStatus = 0
	StatusAuction MarketStatus = 1
	StatusTrading MarketStatus = 2
	StatusHalted  MarketStatus = 3
	StatusClosed  MarketStatus = 4
)

// EngineOrderLevel 表示同一价格档位下的订单集合，保证时间优先 (FIFO)
type EngineOrderLevel struct {
	Price  decimal.Decimal
	Orders *list.List // 存储 *types.Order
}

func NewEngineOrderLevel(price decimal.Decimal) *EngineOrderLevel {
	return &EngineOrderLevel{
		Price:  price,
		Orders: list.New(),
	}
}

// EngineOrderBook 内存订单簿实现
type EngineOrderBook struct {
	Symbol       string
	Bids         *algorithm.SkipList[float64, *EngineOrderLevel]
	Asks         *algorithm.SkipList[float64, *EngineOrderLevel]
	PeggedOrders map[string]*types.Order
	OrderIndex   map[string]*OrderIndexEntry // O(1) OrderID -> Entry
}

type OrderIndexEntry struct {
	Element *list.Element
	Level   *EngineOrderLevel
}

func NewEngineOrderBook(symbol string) *EngineOrderBook {
	return &EngineOrderBook{
		Symbol:       symbol,
		Bids:         algorithm.NewSkipList[float64, *EngineOrderLevel](),
		Asks:         algorithm.NewSkipList[float64, *EngineOrderLevel](),
		PeggedOrders: make(map[string]*types.Order),
		OrderIndex:   make(map[string]*OrderIndexEntry),
	}
}

type EngineMatchTaskType int

const (
	TaskMatch   EngineMatchTaskType = 1
	TaskCancel  EngineMatchTaskType = 2
	TaskAuction EngineMatchTaskType = 3
)

// EngineMatchTask 定义了定序队列中的任务单元
type EngineMatchTask struct {
	Type       EngineMatchTaskType
	Order      *types.Order
	CancelReq  *CancelRequest
	AuctionReq *AuctionRequest
	ResultChan chan any // 改为 any 以兼容不同结果类型
}

type CancelRequest struct {
	OrderID   string
	Symbol    string
	Side      types.Side
	Timestamp int64
}

type AuctionRequest struct {
	Symbol string
}

type CancelResult struct {
	OrderID string
	Success bool
	Status  string
}

// MatchingEngine 核心撮合引擎
type MatchingEngine struct {
	symbol         string
	orderBook      *EngineOrderBook
	ring           *algorithm.MpscRingBuffer[EngineMatchTask]
	stopChan       chan struct{}
	logger         *slog.Logger
	halted         int32
	status         int32           // MarketStatus
	lastPrice      atomic.Value    // decimal.Decimal
	priceCage      decimal.Decimal // 价格笼子比例
	circuitBreaker *CircuitBreaker
}

var taskPool = sync.Pool{
	New: func() any {
		return &EngineMatchTask{
			ResultChan: make(chan any, 1),
		}
	},
}

func NewMatchingEngine(symbol string, capacity uint64, logger *slog.Logger) (*MatchingEngine, error) {
	if logger == nil {
		logger = slog.Default().With("module", "matching_engine", "symbol", symbol)
	}
	ring, err := algorithm.NewMpscRingBuffer[EngineMatchTask](capacity)
	if err != nil {
		return nil, fmt.Errorf("failed to create ring buffer: %w", err)
	}
	engine := &MatchingEngine{
		symbol:    symbol,
		orderBook: NewEngineOrderBook(symbol),
		ring:      ring,
		stopChan:  make(chan struct{}),
		logger:    logger,
		halted:    0,
		status:    int32(StatusInit),
		// 默认 2% 价格笼子
		priceCage: decimal.NewFromFloat(0.02),
		// 默认 10% 熔断阈值，60秒冷却
		circuitBreaker: NewCircuitBreaker(decimal.NewFromFloat(0.10), 60*time.Second, logger),
	}
	engine.lastPrice.Store(decimal.Zero)
	return engine, nil
}

func (e *MatchingEngine) Start() error {
	go e.run()
	return nil
}

func (e *MatchingEngine) Shutdown() {
	close(e.stopChan)
}

func (e *MatchingEngine) IsHalted() bool {
	return atomic.LoadInt32(&e.halted) == 1
}

func (e *MatchingEngine) Halt() {
	atomic.StoreInt32(&e.halted, 1)
}

func (e *MatchingEngine) Resume() {
	e.circuitBreaker.Reset()
	atomic.StoreInt32(&e.halted, 0)
	atomic.StoreInt32(&e.status, int32(StatusTrading))
	e.logger.Info("matching engine resumed")
}

func (e *MatchingEngine) GetStatus() MarketStatus {
	return MarketStatus(atomic.LoadInt32(&e.status))
}

func (e *MatchingEngine) SetStatus(status MarketStatus) {
	atomic.StoreInt32(&e.status, int32(status))
	e.logger.Info("market status changed", "status", status)
}

func (e *MatchingEngine) SetBasePrice(price decimal.Decimal) {
	e.lastPrice.Store(price)
	e.logger.Info("base price set", "price", price)
}

func (e *MatchingEngine) validatePriceCage(price decimal.Decimal) bool {
	last := e.lastPrice.Load().(decimal.Decimal)
	if last.IsZero() {
		return true // 如果没有上一次成交价，暂时跳过校验
	}
	upper := last.Mul(decimal.NewFromInt(1).Add(e.priceCage))
	lower := last.Mul(decimal.NewFromInt(1).Sub(e.priceCage))
	return price.GreaterThanOrEqual(lower) && price.LessThanOrEqual(upper)
}

func (e *MatchingEngine) Symbol() string {
	return e.symbol
}

func (e *MatchingEngine) ReplayOrder(order *types.Order) {
	ob := e.orderBook
	if order.Side == "BUY" {
		e.addToOrderBook(order, ob.Bids, -order.Price.InexactFloat64())
	} else {
		e.addToOrderBook(order, ob.Asks, order.Price.InexactFloat64())
	}
}

func (e *MatchingEngine) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for {
		select {
		case <-e.stopChan:
			return
		default:
			if e.IsHalted() {
				time.Sleep(time.Second)
				continue
			}
			task := e.ring.Poll()
			if task == nil {
				runtime.Gosched()
				continue
			}

			switch task.Type {
			case TaskMatch:
				result := e.applyOrder(task.Order)
				task.ResultChan <- result
			case TaskCancel:
				result := e.processCancel(task.CancelReq)
				task.ResultChan <- result
			case TaskAuction:
				result := e.processAuction(task.AuctionReq)
				task.ResultChan <- result
			}
		}
	}
}

func (e *MatchingEngine) processCancel(req *CancelRequest) *CancelResult {
	res := &CancelResult{OrderID: req.OrderID, Success: false, Status: "ORDER_NOT_FOUND"}

	// 从订单簿中查找并删除
	// 注意：removeFromOrderBook 需要价格，如果没传价格可能需要全表扫描或者引入索引。
	// 为了 Disruptor 模式性能，建议撤单带上价格。
	// 这里假设我们在 application 层已经获取了价格或者对 SkipList 做了符号匹配。
	// 简化处理：遍历该方向的所有档位（仅用于演示，实际应有 Map[OrderID]Price 缓存）

	found := e.removeFromOrderBookByID(req.OrderID, req.Side)
	if found {
		res.Success = true
		res.Status = "CANCELLED"
		e.logger.Info("order cancelled via disruption engine", "order_id", req.OrderID)
	}

	return res
}

func (e *MatchingEngine) processAuction(req *AuctionRequest) *AuctionResult {
	e.logger.Info("executing auction", "symbol", req.Symbol)
	// 初始化拍卖引擎，复用当前订单簿状态
	ae := NewAuctionEngine(e.symbol, decimal.NewFromFloat(0.01), e.logger)
	ae.Bids = e.orderBook.Bids
	ae.Asks = e.orderBook.Asks

	res, err := ae.Match()
	if err != nil {
		e.logger.Warn("auction failed", "error", err)
		return &AuctionResult{}
	}

	// 更新最新成交价
	if res.MatchedQuantity.IsPositive() {
		e.lastPrice.Store(res.EquilibriumPrice)
		// 实际匹配后需要清理订单簿中的成交量。AE.Match() 内部需要处理订单簿扣减。
	}

	return res
}

func (e *MatchingEngine) SubmitOrder(order *types.Order) (*MatchingResult, error) {
	task := taskPool.Get().(*EngineMatchTask)
	task.Type = TaskMatch
	task.Order = order
	defer taskPool.Put(task)

	if !e.ring.Offer(task) {
		return nil, fmt.Errorf("queue full")
	}
	res := <-task.ResultChan
	return res.(*MatchingResult), nil
}

func (e *MatchingEngine) CancelOrder(req *CancelRequest) (*CancelResult, error) {
	task := taskPool.Get().(*EngineMatchTask)
	task.Type = TaskCancel
	task.CancelReq = req
	defer taskPool.Put(task)

	if !e.ring.Offer(task) {
		return nil, fmt.Errorf("queue full")
	}
	res := <-task.ResultChan
	return res.(*CancelResult), nil
}

func (e *MatchingEngine) ExecuteAuction() (*AuctionResult, error) {
	task := taskPool.Get().(*EngineMatchTask)
	task.Type = TaskAuction
	task.AuctionReq = &AuctionRequest{Symbol: e.symbol}
	defer taskPool.Put(task)

	if !e.ring.Offer(task) {
		return nil, fmt.Errorf("queue full")
	}
	res := <-task.ResultChan
	return res.(*AuctionResult), nil
}

func (e *MatchingEngine) applyOrder(order *types.Order) *MatchingResult {
	ob := e.orderBook
	e.repricePeggedOrders(order.Symbol)

	result := &MatchingResult{
		OrderID:           order.OrderID,
		RemainingQuantity: order.Quantity,
		Status:            "PENDING",
	}

	// 1. 状态检查
	if e.GetStatus() != StatusTrading {
		result.Status = "REJECTED_MARKET_CLOSED"
		return result
	}

	// 2. 价格笼子校验
	if !e.validatePriceCage(order.Price) {
		result.Status = "REJECTED_PRICE_OUT_OF_CAGE"
		return result
	}

	// 3. 预查 (针对 FOK/AON)
	var opponentBook *algorithm.SkipList[float64, *EngineOrderLevel]
	if order.Side == "BUY" {
		opponentBook = ob.Asks
	} else {
		opponentBook = ob.Bids
	}

	if order.TimeInForce == types.TIFFOK || order.Condition == types.CondAON {
		possibleQty := e.probeMatchableQuantity(order, opponentBook)
		if order.TimeInForce == types.TIFFOK && possibleQty.LessThan(order.Quantity) {
			result.Status = "CANCELLED_FOK_NOT_FILLED"
			return result
		}
		if order.Condition == types.CondAON && possibleQty.LessThan(order.Quantity) {
			result.Status = "REJECTED_AON_INSUFFICIENT"
			return result
		}
	}

	if order.Side == "BUY" {
		// PostOnly 检查
		if order.PostOnly {
			it := ob.Asks.Iterator()
			if key, _, ok := it.Next(); ok && order.Price.GreaterThanOrEqual(decimal.NewFromFloat(key)) {
				result.Status = "REJECTED_POST_ONLY"
				return result
			}
		}
		e.matchOrder(order, ob.Asks, result)
		// FAK 处理：不加入订单簿，剩余直接撤销
		if result.RemainingQuantity.IsPositive() && order.TimeInForce != types.TIFFAK {
			e.addToOrderBook(order, ob.Bids, -order.Price.InexactFloat64())
		} else if result.RemainingQuantity.IsPositive() && order.TimeInForce == types.TIFFAK {
			result.Status = "CANCELLED_FAK_REMAINDER"
		}
	} else {
		// PostOnly 检查
		if order.PostOnly {
			it := ob.Bids.Iterator()
			if key, _, ok := it.Next(); ok && order.Price.LessThanOrEqual(decimal.NewFromFloat(-key)) {
				result.Status = "REJECTED_POST_ONLY"
				return result
			}
		}
		e.matchOrder(order, ob.Bids, result)
		// FAK 处理：不加入订单簿，剩余直接撤销
		if result.RemainingQuantity.IsPositive() && order.TimeInForce != types.TIFFAK {
			e.addToOrderBook(order, ob.Asks, order.Price.InexactFloat64())
		} else if result.RemainingQuantity.IsPositive() && order.TimeInForce == types.TIFFAK {
			result.Status = "CANCELLED_FAK_REMAINDER"
		}
	}

	if result.Status == "PENDING" || result.Status == "" {
		if len(result.Trades) > 0 {
			if result.RemainingQuantity.IsZero() {
				result.Status = "MATCHED"
			} else {
				result.Status = "PARTIALLY_MATCHED"
			}
		} else if result.RemainingQuantity.IsPositive() {
			result.Status = "NEW"
		}
	}
	return result
}

// probeMatchableQuantity 探测可成交数量（不产生实际成交）
func (e *MatchingEngine) probeMatchableQuantity(order *types.Order, opponentBook *algorithm.SkipList[float64, *EngineOrderLevel]) decimal.Decimal {
	totalPossible := decimal.Zero
	remainingToProbe := order.Quantity

	it := opponentBook.Iterator()
	for {
		_, oppLevel, ok := it.Next()
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

		for el := oppLevel.Orders.Front(); el != nil; el = el.Next() {
			oppOrder := el.Value.(*types.Order)
			availableQty := oppOrder.Quantity
			if oppOrder.IsIceberg {
				availableQty = oppOrder.DisplayQty
				if availableQty.IsZero() && oppOrder.HiddenQty.IsPositive() {
					// 模拟刷新
					refreshAmount := decimal.Min(oppOrder.HiddenQty, oppOrder.Quantity.Mul(decimal.NewFromFloat(0.1)))
					if refreshAmount.IsZero() {
						refreshAmount = oppOrder.HiddenQty
					}
					availableQty = refreshAmount
				}
			}

			if availableQty.IsZero() {
				continue
			}

			matchQty := decimal.Min(remainingToProbe, availableQty)
			totalPossible = totalPossible.Add(matchQty)
			remainingToProbe = remainingToProbe.Sub(matchQty)

			if remainingToProbe.IsZero() {
				return totalPossible
			}
		}
	}
	return totalPossible
}

func (e *MatchingEngine) matchOrder(order *types.Order, opponentBook *algorithm.SkipList[float64, *EngineOrderLevel], result *MatchingResult) {
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
			oppOrder := el.Value.(*types.Order)

			availableQty := oppOrder.Quantity
			if oppOrder.IsIceberg {
				availableQty = oppOrder.DisplayQty
				if availableQty.IsZero() && oppOrder.HiddenQty.IsPositive() {
					e.refreshIceberg(oppOrder)
					availableQty = oppOrder.DisplayQty
				}
			}

			if availableQty.IsZero() {
				continue
			}

			matchQty := decimal.Min(result.RemainingQuantity, availableQty)
			trade := &types.Trade{
				TradeID:   generateEngineTradeID(),
				Symbol:    e.symbol,
				Price:     realOppPrice,
				Quantity:  matchQty,
				Timestamp: time.Now().UnixNano(),
			}

			// 熔断检查
			if !e.circuitBreaker.CheckPrice(realOppPrice) {
				e.Halt()
				e.logger.Error("matching engine halted due to circuit breaker trigger", "price", realOppPrice)
				break // 停止匹配，引擎 Halt 后主循环会暂停处理
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
			e.lastPrice.Store(realOppPrice) // 更新最新成交价

			if oppOrder.Quantity.IsZero() {
				oppLevel.Orders.Remove(el)
				delete(e.orderBook.PeggedOrders, oppOrder.OrderID)
				delete(e.orderBook.OrderIndex, oppOrder.OrderID)
			} else if oppOrder.IsIceberg {
				oppOrder.DisplayQty = oppOrder.DisplayQty.Sub(matchQty)
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

func (e *MatchingEngine) addToOrderBook(order *types.Order, book *algorithm.SkipList[float64, *EngineOrderLevel], key float64) {
	level, ok := book.Search(key)
	if !ok {
		level = NewEngineOrderLevel(order.Price)
		book.Insert(key, level)
	}

	orderCopy := *order
	if orderCopy.IsIceberg && orderCopy.DisplayQty.IsZero() {
		e.refreshIceberg(&orderCopy)
	}
	if orderCopy.IsPegged {
		e.orderBook.PeggedOrders[order.OrderID] = &orderCopy
	}
	el := level.Orders.PushBack(&orderCopy)
	e.orderBook.OrderIndex[order.OrderID] = &OrderIndexEntry{
		Element: el,
		Level:   level,
	}
}

func (e *MatchingEngine) refreshIceberg(order *types.Order) {
	refreshAmount := decimal.Min(order.HiddenQty, order.Quantity.Mul(decimal.NewFromFloat(0.1)))
	if refreshAmount.IsZero() && order.HiddenQty.IsPositive() {
		refreshAmount = order.HiddenQty
	}
	order.DisplayQty = refreshAmount
	order.HiddenQty = order.HiddenQty.Sub(refreshAmount)
}

func (e *MatchingEngine) repricePeggedOrders(symbol string) {
	ob := e.orderBook
	if len(ob.PeggedOrders) == 0 {
		return
	}
	bestBid := decimal.Zero
	bestAsk := decimal.Zero
	itB := ob.Bids.Iterator()
	if _, lv, ok := itB.Next(); ok {
		bestBid = lv.Price
	}
	itA := ob.Asks.Iterator()
	if _, lv, ok := itA.Next(); ok {
		bestAsk = lv.Price
	}

	for _, order := range ob.PeggedOrders {
		var newPrice decimal.Decimal
		switch order.PegType {
		case "MID":
			if !bestBid.IsZero() && !bestAsk.IsZero() {
				newPrice = bestBid.Add(bestAsk).Div(decimal.NewFromInt(2))
			}
		case "BEST_BID":
			newPrice = bestBid.Add(order.PegOffset)
		case "BEST_ASK":
			newPrice = bestAsk.Sub(order.PegOffset)
		}

		if !newPrice.IsZero() && !newPrice.Equal(order.Price) {
			e.removeFromOrderBookByID(order.OrderID, order.Side)
			order.Price = newPrice
			// Re-apply the order to add it back to the order book with the new price
			// and potentially match it if it becomes aggressive.
			e.applyOrder(order)
		}
	}
}

// BatchMatch 批量撮合任务 (用于处理大规模同时到达的请求)
func (e *MatchingEngine) BatchMatch(orders []*types.Order) []*MatchingResult {
	results := make([]*MatchingResult, len(orders))
	for i, order := range orders {
		results[i] = e.applyOrder(order)
	}
	return results
}

func (e *MatchingEngine) removeFromOrderBookByID(orderID string, side types.Side) bool {
	ob := e.orderBook
	entry, ok := ob.OrderIndex[orderID]
	if !ok {
		return false
	}

	// 直接从 entry 获取 level 和 element，无需 Search
	entry.Level.Orders.Remove(entry.Element)

	// 如果该价格档位空了，才需要从 SkipList 删除
	if entry.Level.Orders.Len() == 0 {
		o := entry.Element.Value.(*types.Order)
		var book *algorithm.SkipList[float64, *EngineOrderLevel]
		var key float64
		if o.Side == types.SideBuy {
			book = ob.Bids
			key = -o.Price.InexactFloat64()
		} else {
			book = ob.Asks
			key = o.Price.InexactFloat64()
		}
		book.Delete(key)
	}

	delete(ob.OrderIndex, orderID)
	delete(ob.PeggedOrders, orderID)
	return true
}

// GetOrderBookSnapshot 获取订单簿快照 (为兼容性保留的原名称)
func (e *MatchingEngine) GetOrderBookSnapshot(depth int) *EngineOrderBookSnapshot {
	return e.GetEngineOrderBookSnapshot(depth)
}

// GetEngineOrderBookSnapshot 获取订单簿快照 (支持深度限制)
func (e *MatchingEngine) GetEngineOrderBookSnapshot(depth int) *EngineOrderBookSnapshot {
	ob := e.orderBook
	snapshot := &EngineOrderBookSnapshot{
		Symbol:    ob.Symbol,
		Timestamp: time.Now().UnixNano(),
	}

	itB := ob.Bids.Iterator()
	for i := 0; depth <= 0 || i < depth; i++ {
		_, lv, ok := itB.Next()
		if !ok {
			break
		}
		var qty decimal.Decimal
		for el := lv.Orders.Front(); el != nil; el = el.Next() {
			qty = qty.Add(el.Value.(*types.Order).Quantity)
		}
		snapshot.Bids = append(snapshot.Bids, &EngineOrderBookLevel{Price: lv.Price, Quantity: qty})
	}

	itA := ob.Asks.Iterator()
	for i := 0; depth <= 0 || i < depth; i++ {
		_, lv, ok := itA.Next()
		if !ok {
			break
		}
		var qty decimal.Decimal
		for el := lv.Orders.Front(); el != nil; el = el.Next() {
			qty = qty.Add(el.Value.(*types.Order).Quantity)
		}
		snapshot.Asks = append(snapshot.Asks, &EngineOrderBookLevel{Price: lv.Price, Quantity: qty})
	}

	return snapshot
}

func generateEngineTradeID() string {
	return fmt.Sprintf("ET-%d", time.Now().UnixNano())
}

type MatchingResult struct {
	OrderID           string
	Trades            []*types.Trade
	RemainingQuantity decimal.Decimal
	Status            string
}

type EngineOrderBookLevel struct {
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
}

type EngineOrderBookSnapshot struct {
	Symbol    string                  `json:"symbol"`
	Bids      []*EngineOrderBookLevel `json:"bids"`
	Asks      []*EngineOrderBookLevel `json:"asks"`
	Timestamp int64                   `json:"timestamp"`
}

// AuctionEngine 拍卖引擎
type AuctionEngine struct {
	Symbol  string
	Bids    *algorithm.SkipList[float64, *EngineOrderLevel]
	Asks    *algorithm.SkipList[float64, *EngineOrderLevel]
	MinTick decimal.Decimal
	Logger  *slog.Logger
}

func NewAuctionEngine(symbol string, minTick decimal.Decimal, logger *slog.Logger) *AuctionEngine {
	return &AuctionEngine{
		Symbol:  symbol,
		Bids:    algorithm.NewSkipList[float64, *EngineOrderLevel](),
		Asks:    algorithm.NewSkipList[float64, *EngineOrderLevel](),
		MinTick: minTick,
		Logger:  logger,
	}
}

type AuctionResult struct {
	EquilibriumPrice decimal.Decimal
	MatchedQuantity  decimal.Decimal
	ImbalanceSide    string
	ImbalanceQty     decimal.Decimal
	Trades           []*types.Trade
}

// SubmitOrder 提交订单到拍卖引擎
func (e *AuctionEngine) SubmitOrder(order *types.Order) {
	var book *algorithm.SkipList[float64, *EngineOrderLevel]
	var key float64

	if order.Side == "BUY" {
		book = e.Bids
		key = -order.Price.InexactFloat64() // Bids sort descending
	} else {
		book = e.Asks
		key = order.Price.InexactFloat64() // Asks sort ascending
	}

	level, ok := book.Search(key)
	if !ok {
		level = NewEngineOrderLevel(order.Price)
		book.Insert(key, level)
	}
	level.Orders.PushBack(order)
}

// CalculateEquilibriumPrice 计算平衡价格 (O(N) 实现)
func (e *AuctionEngine) CalculateEquilibriumPrice() (*AuctionResult, error) {
	// 1. 获取所有独立价格点并排序
	priceSet := make(map[string]decimal.Decimal)
	var prices []decimal.Decimal

	itB := e.Bids.Iterator()
	for {
		_, lv, ok := itB.Next()
		if !ok {
			break
		}
		if _, exists := priceSet[lv.Price.String()]; !exists {
			priceSet[lv.Price.String()] = lv.Price
			prices = append(prices, lv.Price)
		}
	}
	itA := e.Asks.Iterator()
	for {
		_, lv, ok := itA.Next()
		if !ok {
			break
		}
		if _, exists := priceSet[lv.Price.String()]; !exists {
			priceSet[lv.Price.String()] = lv.Price
			prices = append(prices, lv.Price)
		}
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("no orders in book")
	}

	sort.Slice(prices, func(i, j int) bool {
		return prices[i].LessThan(prices[j])
	})

	// 2. 计算每个价格级别的累计买量和卖量 (O(N))
	buyVolumes := make([]decimal.Decimal, len(prices))
	sellVolumes := make([]decimal.Decimal, len(prices))

	// 累计买量 (从高价向低价累加)
	totalBuy := decimal.Zero
	currentPriceIdx := len(prices) - 1
	itB = e.Bids.Iterator()
	for {
		_, lv, ok := itB.Next()
		if !ok {
			break
		}
		// 累加当前档位
		levelQty := decimal.Zero
		for el := lv.Orders.Front(); el != nil; el = el.Next() {
			levelQty = levelQty.Add(el.Value.(*types.Order).Quantity)
		}
		totalBuy = totalBuy.Add(levelQty)

		// 更新对应价格点
		for currentPriceIdx >= 0 && prices[currentPriceIdx].GreaterThanOrEqual(lv.Price) {
			buyVolumes[currentPriceIdx] = totalBuy
			currentPriceIdx--
		}
	}
	// 补齐低价档位的累计买量
	for i := currentPriceIdx; i >= 0; i-- {
		buyVolumes[i] = totalBuy
	}

	// 累计卖量 (从低价向高价累加)
	totalSell := decimal.Zero
	currentPriceIdx = 0
	itA = e.Asks.Iterator()
	for {
		_, lv, ok := itA.Next()
		if !ok {
			break
		}
		levelQty := decimal.Zero
		for el := lv.Orders.Front(); el != nil; el = el.Next() {
			levelQty = levelQty.Add(el.Value.(*types.Order).Quantity)
		}
		totalSell = totalSell.Add(levelQty)

		for currentPriceIdx < len(prices) && prices[currentPriceIdx].LessThanOrEqual(lv.Price) {
			sellVolumes[currentPriceIdx] = totalSell
			currentPriceIdx++
		}
	}
	for i := currentPriceIdx; i < len(prices); i++ {
		sellVolumes[i] = totalSell
	}

	// 3. 寻找最大成交量的价格
	var bestPrice decimal.Decimal
	var maxVol decimal.Decimal
	minImbalance := decimal.NewFromInt(1 << 60)

	for i := 0; i < len(prices); i++ {
		matched := decimal.Min(buyVolumes[i], sellVolumes[i])
		if matched.GreaterThan(maxVol) {
			maxVol = matched
			bestPrice = prices[i]
			minImbalance = buyVolumes[i].Sub(sellVolumes[i]).Abs()
		} else if matched.Equal(maxVol) && maxVol.IsPositive() {
			imbalance := buyVolumes[i].Sub(sellVolumes[i]).Abs()
			if imbalance.LessThan(minImbalance) {
				minImbalance = imbalance
				bestPrice = prices[i]
			}
		}
	}

	if maxVol.IsZero() {
		return &AuctionResult{}, nil
	}

	// 4. 生成虚拟交易（实际撮合由调用者处理或进一步细化）
	return &AuctionResult{
		EquilibriumPrice: bestPrice,
		MatchedQuantity:  maxVol,
	}, nil
}

// Match 执行拍卖撮合
func (e *AuctionEngine) Match() (*AuctionResult, error) {
	res, err := e.CalculateEquilibriumPrice()
	if err != nil {
		return nil, err
	}

	ep := res.EquilibriumPrice
	e.Logger.Info("auction equilibrium price calculated", "ep", ep, "vol", res.MatchedQuantity)

	// 收集所有可成交订单
	var buyOrders []*types.Order
	var sellOrders []*types.Order

	itB := e.Bids.Iterator()
	for {
		_, lv, ok := itB.Next()
		if !ok {
			break
		}
		if lv.Price.GreaterThanOrEqual(ep) {
			for el := lv.Orders.Front(); el != nil; el = el.Next() {
				buyOrders = append(buyOrders, el.Value.(*types.Order))
			}
		}
	}

	itA := e.Asks.Iterator()
	for {
		_, lv, ok := itA.Next()
		if !ok {
			break
		}
		if lv.Price.LessThanOrEqual(ep) {
			for el := lv.Orders.Front(); el != nil; el = el.Next() {
				sellOrders = append(sellOrders, el.Value.(*types.Order))
			}
		}
	}

	// 价格优先，时间优先匹配
	bIdx, sIdx := 0, 0
	for bIdx < len(buyOrders) && sIdx < len(sellOrders) {
		buyOrd := buyOrders[bIdx]
		sellOrd := sellOrders[sIdx]
		qty := decimal.Min(buyOrd.Quantity, sellOrd.Quantity)
		if qty.IsZero() {
			if buyOrd.Quantity.IsZero() {
				bIdx++
			}
			if sellOrd.Quantity.IsZero() {
				sIdx++
			}
			continue
		}
		trade := &types.Trade{
			TradeID:     generateEngineTradeID(),
			Symbol:      e.Symbol,
			Price:       ep,
			Quantity:    qty,
			BuyOrderID:  buyOrd.OrderID,
			SellOrderID: sellOrd.OrderID,
			Timestamp:   time.Now().UnixNano(),
		}
		res.Trades = append(res.Trades, trade)
		buyOrd.Quantity = buyOrd.Quantity.Sub(qty)
		sellOrd.Quantity = sellOrd.Quantity.Sub(qty)
		if buyOrd.Quantity.IsZero() {
			bIdx++
		}
		if sellOrd.Quantity.IsZero() {
			sIdx++
		}
	}
	return res, nil
}

// MarketData 市场数据聚合
type MarketData struct {
	Bid float64
	Ask float64
}

// OrderBookDepth 订单簿深度数据
type OrderBookDepth struct {
	Bids []*PriceLevel
	Asks []*PriceLevel
}

func (e *MatchingEngine) GetMarketData() MarketData {
	bestBid := decimal.Zero
	bestAsk := decimal.Zero

	if e.orderBook.Bids != nil {
		if _, level, ok := e.orderBook.Bids.Iterator().Next(); ok {
			bestBid = level.Price
		}
	}

	if e.orderBook.Asks != nil {
		if _, level, ok := e.orderBook.Asks.Iterator().Next(); ok {
			bestAsk = level.Price
		}
	}

	return MarketData{
		Bid: bestBid.InexactFloat64(),
		Ask: bestAsk.InexactFloat64(),
	}
}

func (e *MatchingEngine) GetOrderBookDepth(depth int) (*OrderBookDepth, error) {
	if depth <= 0 {
		depth = 10
	}

	snapshot := e.GetEngineOrderBookSnapshot(depth)

	bids := make([]*PriceLevel, len(snapshot.Bids))
	for i, b := range snapshot.Bids {
		bids[i] = &PriceLevel{
			Price:    b.Price,
			Quantity: b.Quantity,
		}
	}

	asks := make([]*PriceLevel, len(snapshot.Asks))
	for i, a := range snapshot.Asks {
		asks[i] = &PriceLevel{
			Price:    a.Price,
			Quantity: a.Quantity,
		}
	}

	return &OrderBookDepth{
		Bids: bids,
		Asks: asks,
	}, nil
}
