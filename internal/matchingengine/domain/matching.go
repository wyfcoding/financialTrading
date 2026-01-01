// Package domain 撮合引擎的领域模型
package domain

import (
	"context"
	"container/list"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algorithm"
	"gorm.io/gorm"
)

// OrderLevel 表示同一价格档位下的订单集合，保证时间优先 (FIFO)
// 这里的命名为了避开 pkg/algorithm/data_structures.go 中的 PriceLevel 冲突
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

// OrderBook 内存订单簿实现
type OrderBook struct {
	mu     sync.RWMutex
	Symbol string

	// Bids 买盘：Key 为 -Price (取反实现降序，高价优先)，Value 为 OrderLevel
	Bids *algorithm.SkipList[float64, *OrderLevel]
	// Asks 卖盘：Key 为 Price (升序，低价优先)，Value 为 OrderLevel
	Asks *algorithm.SkipList[float64, *OrderLevel]
}

func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		Symbol: symbol,
		Bids:   algorithm.NewSkipList[float64, *OrderLevel](),
		Asks:   algorithm.NewSkipList[float64, *OrderLevel](),
	}
}

// ApplyOrder 提交订单到引擎进行撮合。
func (ob *OrderBook) ApplyOrder(order *algorithm.Order) *MatchingResult {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	result := &MatchingResult{
		OrderID:           order.OrderID,
		RemainingQuantity: order.Quantity,
		Status:            "PENDING",
	}

	if order.Side == "BUY" {
		ob.matchOrder(order, ob.Asks, result)
		if result.RemainingQuantity.IsPositive() {
			ob.addToOrderBook(order, ob.Bids, -order.Price.InexactFloat64())
		}
	} else {
		ob.matchOrder(order, ob.Bids, result)
		if result.RemainingQuantity.IsPositive() {
			ob.addToOrderBook(order, ob.Asks, order.Price.InexactFloat64())
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

func (ob *OrderBook) matchOrder(order *algorithm.Order, opponentBook *algorithm.SkipList[float64, *OrderLevel], result *MatchingResult) {
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
		for e := oppLevel.Orders.Front(); e != nil; e = nextOrder {
			nextOrder = e.Next()
			oppOrder := e.Value.(*algorithm.Order)

			matchQty := decimal.Min(result.RemainingQuantity, oppOrder.Quantity)
			
			trade := &algorithm.Trade{
				TradeID:    generateTradeID(),
				Symbol:     ob.Symbol,
				Price:      realOppPrice,
				Quantity:   matchQty,
				Timestamp:  time.Now().UnixNano(),
			}

			// 对齐 pkg/algorithm/orderbook.go 中的字段
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
				oppLevel.Orders.Remove(e)
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

func (ob *OrderBook) addToOrderBook(order *algorithm.Order, book *algorithm.SkipList[float64, *OrderLevel], key float64) {
	level, ok := book.Search(key)
	if !ok {
		level = NewOrderLevel(order.Price)
		book.Insert(key, level)
	}
	orderCopy := *order
	orderCopy.Quantity = order.Quantity
	level.Orders.PushBack(&orderCopy)
}

// GetDepth 获取订单簿指定深度的档位信息
func (ob *OrderBook) GetDepth(depth int) ([]*OrderBookLevel, []*OrderBookLevel) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids := ob.collectLevels(ob.Bids, depth)
	asks := ob.collectLevels(ob.Asks, depth)
	return bids, asks
}

func (ob *OrderBook) collectLevels(book *algorithm.SkipList[float64, *OrderLevel], depth int) []*OrderBookLevel {
	levels := make([]*OrderBookLevel, 0, depth)
	it := book.Iterator()
	for i := 0; i < depth; i++ {
		_, level, ok := it.Next()
		if !ok {
			break
		}
		
		var totalQty decimal.Decimal
		for e := level.Orders.Front(); e != nil; e = e.Next() {
			totalQty = totalQty.Add(e.Value.(*algorithm.Order).Quantity)
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

// OrderBookLevel 订单簿档位（用于展示、快照和 API 返回）
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

// MatchingEngine 撮合引擎接口
type MatchingEngine interface {
	SubmitOrder(order *algorithm.Order) *MatchingResult
	GetOrderBook(symbol string, depth int) *OrderBookSnapshot
	GetTrades(symbol string, limit int) []*algorithm.Trade
}

// TradeModel 成交记录仓储模型
type TradeModel struct {
	gorm.Model
	TradeID      string          `gorm:"column:trade_id;type:varchar(32);uniqueIndex;not null"`
	OrderID      string          `gorm:"column:order_id;type:varchar(32);index;not null"`
	MatchOrderID string          `gorm:"column:match_order_id;type:varchar(32);index;not null"`
	Symbol       string          `gorm:"column:symbol;type:varchar(20);index;not null"`
	Side         string          `gorm:"column:side;type:varchar(10);not null"`
	Price        decimal.Decimal `gorm:"column:price;type:decimal(32,18);not null"`
	Quantity     decimal.Decimal `gorm:"column:quantity;type:decimal(32,18);not null"`
	ExecutedAt   int64           `gorm:"column:executed_at;type:bigint"`
}

// TradeRepository 成交记录仓储接口
type TradeRepository interface {
	Save(ctx context.Context, trade *algorithm.Trade) error
	GetTradeHistory(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error)
	GetLatestTrades(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error)
}

// OrderBookRepository 订单簿仓储接口
type OrderBookRepository interface {
	SaveSnapshot(ctx context.Context, snapshot *OrderBookSnapshot) error
	GetLatestOrderBook(ctx context.Context, symbol string) (*OrderBookSnapshot, error)
}
