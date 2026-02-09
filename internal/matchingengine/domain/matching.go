package domain

import (
	"container/list"
	"fmt"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"

	algorithm "github.com/wyfcoding/pkg/algorithm/structures"
	"github.com/wyfcoding/pkg/algorithm/types"

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

// OrderLevel 表示同一价格档位下的订单集合，保证时间优先 (FIFO)
type OrderLevel struct {
	Price  decimal.Decimal
	Orders *list.List // 存储 *types.Order
}

func NewOrderLevel(price decimal.Decimal) *OrderLevel {
	return &OrderLevel{
		Price:  price,
		Orders: list.New(),
	}
}

// OrderBook 内存订单簿实现
type OrderBook struct {
	Symbol       string
	Bids         *algorithm.SkipList[float64, *OrderLevel]
	Asks         *algorithm.SkipList[float64, *OrderLevel]
	PeggedOrders map[string]*types.Order
}

func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		Symbol:       symbol,
		Bids:         algorithm.NewSkipList[float64, *OrderLevel](),
		Asks:         algorithm.NewSkipList[float64, *OrderLevel](),
		PeggedOrders: make(map[string]*types.Order),
	}
}

type MatchTaskType int

const (
	TaskMatch   MatchTaskType = 1
	TaskCancel  MatchTaskType = 2
	TaskAuction MatchTaskType = 3
)

// MatchTask 定义了定序队列中的任务单元
type MatchTask struct {
	Type       MatchTaskType
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

// DisruptionEngine 核心撮合引擎
type DisruptionEngine struct {
	symbol         string
	orderBook      *OrderBook
	ring           *algorithm.MpscRingBuffer[MatchTask]
	stopChan       chan struct{}
	logger         *slog.Logger
	halted         int32
	status         int32           // MarketStatus
	lastPrice      atomic.Value    // decimal.Decimal
	priceCage      decimal.Decimal // 价格笼子比例
	circuitBreaker *CircuitBreaker
}

func NewDisruptionEngine(symbol string, capacity uint64, logger *slog.Logger) (*DisruptionEngine, error) {
	if logger == nil {
		logger = slog.Default().With("module", "disruption_engine", "symbol", symbol)
	}
	ring, err := algorithm.NewMpscRingBuffer[MatchTask](capacity)
	if err != nil {
		return nil, fmt.Errorf("failed to create ring buffer: %w", err)
	}
	engine := &DisruptionEngine{
		symbol:    symbol,
		orderBook: NewOrderBook(symbol),
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

func (e *DisruptionEngine) Start() error {
	go e.run()
	return nil
}

func (e *DisruptionEngine) Shutdown() {
	close(e.stopChan)
}

func (e *DisruptionEngine) IsHalted() bool {
	return atomic.LoadInt32(&e.halted) == 1
}

func (e *DisruptionEngine) Halt() {
	atomic.StoreInt32(&e.halted, 1)
}

func (e *DisruptionEngine) Resume() {
	e.circuitBreaker.Reset()
	atomic.StoreInt32(&e.halted, 0)
	atomic.StoreInt32(&e.status, int32(StatusTrading))
	e.logger.Info("matching engine resumed")
}

func (e *DisruptionEngine) GetStatus() MarketStatus {
	return MarketStatus(atomic.LoadInt32(&e.status))
}

func (e *DisruptionEngine) SetStatus(status MarketStatus) {
	atomic.StoreInt32(&e.status, int32(status))
	e.logger.Info("market status changed", "status", status)
}

func (e *DisruptionEngine) SetBasePrice(price decimal.Decimal) {
	e.lastPrice.Store(price)
	e.logger.Info("base price set", "price", price)
}

func (e *DisruptionEngine) validatePriceCage(price decimal.Decimal) bool {
	last := e.lastPrice.Load().(decimal.Decimal)
	if last.IsZero() {
		return true // 如果没有上一次成交价，暂时跳过校验
	}
	upper := last.Mul(decimal.NewFromInt(1).Add(e.priceCage))
	lower := last.Mul(decimal.NewFromInt(1).Sub(e.priceCage))
	return price.GreaterThanOrEqual(lower) && price.LessThanOrEqual(upper)
}

func (e *DisruptionEngine) Symbol() string {
	return e.symbol
}

func (e *DisruptionEngine) ReplayOrder(order *types.Order) {
	ob := e.orderBook
	if order.Side == "BUY" {
		e.addToOrderBook(order, ob.Bids, -order.Price.InexactFloat64())
	} else {
		e.addToOrderBook(order, ob.Asks, order.Price.InexactFloat64())
	}
}

func (e *DisruptionEngine) run() {
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

func (e *DisruptionEngine) processCancel(req *CancelRequest) *CancelResult {
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

func (e *DisruptionEngine) processAuction(req *AuctionRequest) *AuctionResult {
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

func (e *DisruptionEngine) SubmitOrder(order *types.Order) (*MatchingResult, error) {
	resChan := make(chan any, 1)
	task := &MatchTask{Type: TaskMatch, Order: order, ResultChan: resChan}
	if !e.ring.Offer(task) {
		return nil, fmt.Errorf("queue full")
	}
	res := <-resChan
	return res.(*MatchingResult), nil
}

func (e *DisruptionEngine) CancelOrder(req *CancelRequest) (*CancelResult, error) {
	resChan := make(chan any, 1)
	task := &MatchTask{Type: TaskCancel, CancelReq: req, ResultChan: resChan}
	if !e.ring.Offer(task) {
		return nil, fmt.Errorf("queue full")
	}
	res := <-resChan
	return res.(*CancelResult), nil
}

func (e *DisruptionEngine) ExecuteAuction() (*AuctionResult, error) {
	resChan := make(chan any, 1)
	task := &MatchTask{Type: TaskAuction, AuctionReq: &AuctionRequest{Symbol: e.symbol}, ResultChan: resChan}
	if !e.ring.Offer(task) {
		return nil, fmt.Errorf("queue full")
	}
	res := <-resChan
	return res.(*AuctionResult), nil
}

func (e *DisruptionEngine) applyOrder(order *types.Order) *MatchingResult {
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
	var opponentBook *algorithm.SkipList[float64, *OrderLevel]
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
func (e *DisruptionEngine) probeMatchableQuantity(order *types.Order, opponentBook *algorithm.SkipList[float64, *OrderLevel]) decimal.Decimal {
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

func (e *DisruptionEngine) matchOrder(order *types.Order, opponentBook *algorithm.SkipList[float64, *OrderLevel], result *MatchingResult) {
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
				TradeID:   generateTradeID(),
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

func (e *DisruptionEngine) addToOrderBook(order *types.Order, book *algorithm.SkipList[float64, *OrderLevel], key float64) {
	level, ok := book.Search(key)
	if !ok {
		level = NewOrderLevel(order.Price)
		book.Insert(key, level)
	}

	orderCopy := *order
	if orderCopy.IsIceberg && orderCopy.DisplayQty.IsZero() {
		e.refreshIceberg(&orderCopy)
	}
	if orderCopy.IsPegged {
		e.orderBook.PeggedOrders[order.OrderID] = &orderCopy
	}
	level.Orders.PushBack(&orderCopy)
}

func (e *DisruptionEngine) refreshIceberg(order *types.Order) {
	refreshAmount := decimal.Min(order.HiddenQty, order.Quantity.Mul(decimal.NewFromFloat(0.1)))
	if refreshAmount.IsZero() && order.HiddenQty.IsPositive() {
		refreshAmount = order.HiddenQty
	}
	order.DisplayQty = refreshAmount
	order.HiddenQty = order.HiddenQty.Sub(refreshAmount)
}

func (e *DisruptionEngine) repricePeggedOrders(symbol string) {
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
func (e *DisruptionEngine) BatchMatch(orders []*types.Order) []*MatchingResult {
	results := make([]*MatchingResult, len(orders))
	for i, order := range orders {
		results[i] = e.applyOrder(order)
	}
	return results
}

func (e *DisruptionEngine) removeFromOrderBookByID(orderID string, side types.Side) bool {
	ob := e.orderBook
	var book *algorithm.SkipList[float64, *OrderLevel]
	if side == types.SideBuy {
		book = ob.Bids
	} else {
		book = ob.Asks
	}

	it := book.Iterator()
	for {
		key, lv, ok := it.Next()
		if !ok {
			break
		}
		for el := lv.Orders.Front(); el != nil; el = el.Next() {
			o := el.Value.(*types.Order)
			if o.OrderID == orderID {
				lv.Orders.Remove(el)
				if lv.Orders.Len() == 0 {
					book.Delete(key)
				}
				delete(ob.PeggedOrders, orderID)
				return true
			}
		}
	}
	return false
}

// GetOrderBookSnapshot 获取订单簿快照 (支持深度限制)
func (e *DisruptionEngine) GetOrderBookSnapshot(depth int) *OrderBookSnapshot {
	ob := e.orderBook
	snapshot := &OrderBookSnapshot{
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
		snapshot.Bids = append(snapshot.Bids, &OrderBookLevel{Price: lv.Price, Quantity: qty})
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
		snapshot.Asks = append(snapshot.Asks, &OrderBookLevel{Price: lv.Price, Quantity: qty})
	}

	return snapshot
}

func generateTradeID() string {
	return fmt.Sprintf("T-%d", time.Now().UnixNano())
}

type MatchingResult struct {
	OrderID           string
	Trades            []*types.Trade
	RemainingQuantity decimal.Decimal
	Status            string
}

type OrderBookLevel struct {
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
}

type OrderBookSnapshot struct {
	ID        uint              `json:"id"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Symbol    string            `json:"symbol"`
	Bids      []*OrderBookLevel `json:"bids"`
	Asks      []*OrderBookLevel `json:"asks"`
	Timestamp int64             `json:"timestamp"`
}

// AuctionEngine 拍卖引擎
type AuctionEngine struct {
	Symbol  string
	Bids    *algorithm.SkipList[float64, *OrderLevel]
	Asks    *algorithm.SkipList[float64, *OrderLevel]
	MinTick decimal.Decimal
	Logger  *slog.Logger
}

func NewAuctionEngine(symbol string, minTick decimal.Decimal, logger *slog.Logger) *AuctionEngine {
	return &AuctionEngine{
		Symbol:  symbol,
		Bids:    algorithm.NewSkipList[float64, *OrderLevel](),
		Asks:    algorithm.NewSkipList[float64, *OrderLevel](),
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
	var book *algorithm.SkipList[float64, *OrderLevel]
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
		level = NewOrderLevel(order.Price)
		book.Insert(key, level)
	}
	level.Orders.PushBack(order)
}

// CalculateEquilibriumPrice 计算平衡价格
func (e *AuctionEngine) CalculateEquilibriumPrice() (*AuctionResult, error) {
	// 1. 收集所有独立价格点
	priceMap := make(map[string]decimal.Decimal)
	var prices []decimal.Decimal

	itB := e.Bids.Iterator()
	for {
		_, lv, ok := itB.Next()
		if !ok {
			break
		}
		if _, exists := priceMap[lv.Price.String()]; !exists {
			priceMap[lv.Price.String()] = lv.Price
			prices = append(prices, lv.Price)
		}
	}

	itA := e.Asks.Iterator()
	for {
		_, lv, ok := itA.Next()
		if !ok {
			break
		}
		if _, exists := priceMap[lv.Price.String()]; !exists {
			priceMap[lv.Price.String()] = lv.Price
			prices = append(prices, lv.Price)
		}
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("no orders in book")
	}

	// 2. 对每个价格点计算累积买单和卖单数量
	var bestPrice decimal.Decimal
	var maxVol decimal.Decimal
	minImbalance := decimal.NewFromInt(1000000000) // Max Value

	// 简单的 O(N^2) 实现，生产环境应优化为 O(N) 累积数组
	// 这里为了准确性遍历计算
	for _, p := range prices {
		// Buy Qty: Sum(Bid.Qty) where Bid.Price >= p
		buyQty := decimal.Zero
		itB := e.Bids.Iterator()
		for {
			_, lv, ok := itB.Next()
			if !ok {
				break
			}
			if lv.Price.GreaterThanOrEqual(p) {
				for el := lv.Orders.Front(); el != nil; el = el.Next() {
					buyQty = buyQty.Add(el.Value.(*types.Order).Quantity)
				}
			}
		}

		// Sell Qty: Sum(Ask.Qty) where Ask.Price <= p
		sellQty := decimal.Zero
		itA := e.Asks.Iterator()
		for {
			_, lv, ok := itA.Next()
			if !ok {
				break
			}
			if lv.Price.LessThanOrEqual(p) {
				for el := lv.Orders.Front(); el != nil; el = el.Next() {
					sellQty = sellQty.Add(el.Value.(*types.Order).Quantity)
				}
			}
		}

		execVol := decimal.Min(buyQty, sellQty)
		imbalance := buyQty.Sub(sellQty).Abs()

		// 3. 选择最大成交量
		if execVol.GreaterThan(maxVol) {
			maxVol = execVol
			bestPrice = p
			minImbalance = imbalance
		} else if execVol.Equal(maxVol) {
			// 4. 最小不平衡量
			if imbalance.LessThan(minImbalance) {
				minImbalance = imbalance
				bestPrice = p
			} else if imbalance.Equal(minImbalance) {
				// 5. 市场压力 (简单取均值或离中间价近的，这里取较高价以促进成交)
				// 实际规则可能更复杂
				if p.GreaterThan(bestPrice) {
					bestPrice = p
				}
			}
		}
	}

	if maxVol.IsZero() {
		return nil, fmt.Errorf("no matchable volume")
	}

	return &AuctionResult{
		EquilibriumPrice: bestPrice,
		MatchedQuantity:  maxVol,
	}, nil
}

// Match 执行撮合
func (e *AuctionEngine) Match() (*AuctionResult, error) {
	res, err := e.CalculateEquilibriumPrice()
	if err != nil {
		return nil, err
	}

	ep := res.EquilibriumPrice
	e.Logger.Info("auction equilibrium price calculated", "ep", ep, "vol", res.MatchedQuantity)

	// 收集所有可成交订单
	// Bids: Price >= EP
	// Asks: Price <= EP
	var buyOrders []*types.Order
	var sellOrders []*types.Order

	// 提取买单 (按价格优先，时间优先)
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

	// 提取卖单 (按价格优先，时间优先)
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

	// 执行匹配
	// 双指针匹配
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
			TradeID:     generateTradeID(),
			Symbol:      e.Symbol,
			Price:       ep, // 统一按 EP 成交
			Quantity:    qty,
			BuyOrderID:  buyOrd.OrderID,
			SellOrderID: sellOrd.OrderID,
			Timestamp:   time.Now().UnixNano(),
		}
		res.Trades = append(res.Trades, trade)

		// 更新订单/Book
		buyOrd.Quantity = buyOrd.Quantity.Sub(qty)
		sellOrd.Quantity = sellOrd.Quantity.Sub(qty)

		if buyOrd.Quantity.IsZero() {
			e.removeOrder(buyOrd) // 辅助函数移除
			bIdx++
		}
		if sellOrd.Quantity.IsZero() {
			e.removeOrder(sellOrd)
			sIdx++
		}
	}

	res.MatchedQuantity = decimal.Zero
	for _, t := range res.Trades {
		res.MatchedQuantity = res.MatchedQuantity.Add(t.Quantity)
	}

	// 计算剩余 Imbalance
	return res, nil
}

func (e *AuctionEngine) removeOrder(order *types.Order) {
	var book *algorithm.SkipList[float64, *OrderLevel]
	key := order.Price.InexactFloat64()
	if order.Side == "BUY" {
		book = e.Bids
		key = -key
	} else {
		book = e.Asks
	}

	if lv, ok := book.Search(key); ok {
		for el := lv.Orders.Front(); el != nil; el = el.Next() {
			if el.Value.(*types.Order).OrderID == order.OrderID {
				lv.Orders.Remove(el)
				if lv.Orders.Len() == 0 {
					book.Delete(key)
				}
				return
			}
		}
	}
}
