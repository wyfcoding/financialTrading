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

// MatchTask 定义了定序队列中的任务单元
type MatchTask struct {
	Order      *types.Order
	ResultChan chan *MatchingResult
}

// DisruptionEngine 核心撮合引擎
type DisruptionEngine struct {
	symbol    string
	orderBook *OrderBook
	ring      *algorithm.MpscRingBuffer[MatchTask]
	stopChan  chan struct{}
	logger    *slog.Logger
	halted    int32
}

func NewDisruptionEngine(symbol string, capacity uint64, logger *slog.Logger) (*DisruptionEngine, error) {
	if logger == nil {
		logger = slog.Default().With("module", "disruption_engine", "symbol", symbol)
	}
	ring, err := algorithm.NewMpscRingBuffer[MatchTask](capacity)
	if err != nil {
		return nil, fmt.Errorf("failed to create ring buffer: %w", err)
	}
	return &DisruptionEngine{
		symbol:    symbol,
		orderBook: NewOrderBook(symbol),
		ring:      ring,
		stopChan:  make(chan struct{}),
		logger:    logger,
		halted:    0,
	}, nil
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
			result := e.applyOrder(task.Order)
			task.ResultChan <- result
		}
	}
}

func (e *DisruptionEngine) SubmitOrder(order *types.Order) (*MatchingResult, error) {
	resChan := make(chan *MatchingResult, 1)
	task := &MatchTask{Order: order, ResultChan: resChan}
	if !e.ring.Offer(task) {
		return nil, fmt.Errorf("queue full")
	}
	return <-resChan, nil
}

func (e *DisruptionEngine) applyOrder(order *types.Order) *MatchingResult {
	ob := e.orderBook
	e.repricePeggedOrders(order.Symbol)

	result := &MatchingResult{
		OrderID:           order.OrderID,
		RemainingQuantity: order.Quantity,
		Status:            "PENDING",
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
		if result.RemainingQuantity.IsPositive() {
			e.addToOrderBook(order, ob.Bids, -order.Price.InexactFloat64())
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
		if result.RemainingQuantity.IsPositive() {
			e.addToOrderBook(order, ob.Asks, order.Price.InexactFloat64())
		}
	}

	if len(result.Trades) > 0 {
		if result.RemainingQuantity.IsZero() {
			result.Status = "MATCHED"
		} else {
			result.Status = "PARTIALLY_MATCHED"
		}
	}
	return result
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
			e.removeFromOrderBook(order)
			order.Price = newPrice
			if order.Side == "BUY" {
				e.addToOrderBook(order, ob.Bids, -order.Price.InexactFloat64())
			} else {
				e.addToOrderBook(order, ob.Asks, order.Price.InexactFloat64())
			}
		}
	}
}

func (e *DisruptionEngine) removeFromOrderBook(order *types.Order) {
	ob := e.orderBook
	var book *algorithm.SkipList[float64, *OrderLevel]
	key := order.Price.InexactFloat64()
	if order.Side == "BUY" {
		book = ob.Bids
		key = -key
	} else {
		book = ob.Asks
	}
	if lv, ok := book.Search(key); ok {
		for el := lv.Orders.Front(); el != nil; el = el.Next() {
			if el.Value.(*types.Order).OrderID == order.OrderID {
				lv.Orders.Remove(el)
				break
			}
		}
		if lv.Orders.Len() == 0 {
			book.Delete(key)
		}
	}
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
	ID        uint             `json:"id"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Symbol    string           `json:"symbol"`
	Bids      []*OrderBookLevel `json:"bids"`
	Asks      []*OrderBookLevel `json:"asks"`
	Timestamp int64            `json:"timestamp"`
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
