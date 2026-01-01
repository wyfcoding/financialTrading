// Package domain 撮合引擎的领域模型
package domain

import (
	"container/list"
	"context"
	"fmt"
	"runtime"
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

// DisruptionEngine 基于 MpscRingBuffer 的高性能撮合引擎
type DisruptionEngine struct {
	symbol    string
	orderBook *OrderBook
	ring      *algorithm.MpscRingBuffer[MatchTask]
	stopChan  chan struct{}
}

func NewDisruptionEngine(symbol string, capacity uint64) (*DisruptionEngine, error) {
	ring, err := algorithm.NewMpscRingBuffer[MatchTask](capacity)
	if err != nil {
		return nil, fmt.Errorf("failed to create ring buffer: %w", err)
	}
	e := &DisruptionEngine{
		symbol:    symbol,
		orderBook: NewOrderBook(symbol),
		ring:      ring,
		stopChan:  make(chan struct{}),
	}
	// 启动核心撮合 Worker (单线程)
	go e.run()
	return e, nil
}

func (e *DisruptionEngine) run() {
	// 将 Worker 绑定到固定操作系统线程以获得极致性能
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for {
		select {
		case <-e.stopChan:
			return
		default:
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

// SubmitOrder 提交订单到引擎
func (e *DisruptionEngine) SubmitOrder(order *algorithm.Order) (*MatchingResult, error) {
	resChan := make(chan *MatchingResult, 1)
	task := &MatchTask{
		Order:      order,
		ResultChan: resChan,
	}

	if !e.ring.Offer(task) {
		return nil, fmt.Errorf("engine busy, ring buffer full")
	}

	return <-resChan, nil
}

// applyOrder 核心内部撮合逻辑
func (e *DisruptionEngine) applyOrder(order *algorithm.Order) *MatchingResult {
	ob := e.orderBook
	result := &MatchingResult{
		OrderID:           order.OrderID,
		RemainingQuantity: order.Quantity,
		Status:            "PENDING",
	}

	if order.Side == "BUY" {
		e.matchOrder(order, ob.Asks, result)
		if result.RemainingQuantity.IsPositive() {
			e.addToOrderBook(order, ob.Bids, -order.Price.InexactFloat64())
		}
	} else {
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

func generateTradeID() string {
	return "T-" + time.Now().Format("20060102150405.000000")
}

// MatchingResult 撮合结果
type MatchingResult struct {
	OrderID           string             // 订单 ID
	Trades            []*algorithm.Trade // 成交列表
	RemainingQuantity decimal.Decimal    // 剩余数量
	Status            string             // 状态 (MATCHED, PARTIALLY_MATCHED)
}

// OrderBookLevel 订单簿档位
type OrderBookLevel struct {
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
}

// OrderBookSnapshot 订单簿快照
type OrderBookSnapshot struct {
	gorm.Model
	Symbol    string            `gorm:"column:symbol;type:varchar(20);index;not null"`
	Bids      []*OrderBookLevel `gorm:"-"`
	Asks      []*OrderBookLevel `gorm:"-"`
	BidsJSON  string            `gorm:"column:bids;type:text"`
	AsksJSON  string            `gorm:"column:asks;type:text"`
	Timestamp int64             `gorm:"column:timestamp;type:bigint"`
}

// TradeRepository 成交记录仓储接口
type TradeRepository interface {
	Save(ctx context.Context, trade *algorithm.Trade) error
	GetLatestTrades(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error)
}

// OrderBookRepository 订单簿仓储接口
type OrderBookRepository interface {
	SaveSnapshot(ctx context.Context, snapshot *OrderBookSnapshot) error
}
