package memory

import (
	"container/list"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
)

// OrderLevel 价格档位(Level 2)
type OrderLevel struct {
	Price    decimal.Decimal
	Orders   *list.List // 存储 *domain.Order
	TotalQty decimal.Decimal
}

func NewOrderLevel(price decimal.Decimal) *OrderLevel {
	return &OrderLevel{
		Price:    price,
		Orders:   list.New(),
		TotalQty: decimal.Zero,
	}
}

// InMemoryOrderBook 简单的内存订单簿实现
type InMemoryOrderBook struct {
	Symbol     string
	Bids       map[string]*OrderLevel   // Price string -> OrderLevel
	BidPrices  []decimal.Decimal        // 有序的买单价格 (降序: High -> Low)
	Asks       map[string]*OrderLevel   // Price string -> OrderLevel
	AskPrices  []decimal.Decimal        // 有序的卖单价格 (升序: Low -> High)
	OrderIndex map[uint64]*list.Element // 快速查找订单: OrderID -> Element
}

func NewOrderBook(symbol string) *InMemoryOrderBook {
	return &InMemoryOrderBook{
		Symbol:     symbol,
		Bids:       make(map[string]*OrderLevel),
		Asks:       make(map[string]*OrderLevel),
		OrderIndex: make(map[uint64]*list.Element),
	}
}

// matchOrder 主要撮合逻辑
func (ob *InMemoryOrderBook) AddOrder(order *domain.Order) ([]*domain.Trade, error) {
	trades := []*domain.Trade{}

	if order.Side == domain.Buy {
		trades = ob.matchBid(order)
	} else {
		trades = ob.matchAsk(order)
	}

	// 如果还有剩余，放入订单簿 (Limit Order)
	if order.Type == domain.Limit && order.Quantity.GreaterThan(decimal.Zero) {
		ob.addRestOrder(order)
	}

	return trades, nil
}

// matchBid 撮合买单：去吃 Ask (Low -> High)
func (ob *InMemoryOrderBook) matchBid(taker *domain.Order) []*domain.Trade {
	var trades []*domain.Trade

	// 遍历卖单价格 (升序)
	for i := 0; i < len(ob.AskPrices); {
		bestAskPrice := ob.AskPrices[i]

		// 价格不满足（买价比卖一低），停止撮合
		if taker.Type == domain.Limit && taker.Price.LessThan(bestAskPrice) {
			break
		}

		level := ob.Asks[bestAskPrice.String()]
		tradeAmount := decimal.Zero

		// 遍历该价格档位的订单 (时间优先)
		var next *list.Element
		for e := level.Orders.Front(); e != nil; e = next {
			next = e.Next()
			maker := e.Value.(*domain.Order)

			// 计算成交量
			tradeQty := decimal.Min(taker.Quantity, maker.Quantity)

			trades = append(trades, &domain.Trade{
				TradeID:      fmt.Sprintf("T-%d-%d-%d", maker.ID, taker.ID, taker.Timestamp),
				BuyOrderID:   fmt.Sprintf("%d", taker.ID),
				SellOrderID:  fmt.Sprintf("%d", maker.ID),
				Symbol:       ob.Symbol,
				Price:        maker.Price.InexactFloat64(), // 价格以 Maker 为准
				Quantity:     tradeQty.InexactFloat64(),
				BuyerID:      taker.ID, // 买方是 Taker (只是暂用 ID 代替 UserID)
				SellerID:     maker.ID, // 卖方是 Maker
				TakerOrder:   taker.ID,
				MakerOrders:  []uint64{maker.ID},
				Timestamp:    time.Unix(0, taker.Timestamp),
				MakerOrderID: maker.ID,
				TakerOrderID: taker.ID,
				Qty:          tradeQty.IntPart(),
				TradeTime:    taker.Timestamp,
			})

			// 更新数量
			maker.Quantity = maker.Quantity.Sub(tradeQty)
			maker.FilledQty = maker.FilledQty.Add(tradeQty)
			taker.Quantity = taker.Quantity.Sub(tradeQty)
			taker.FilledQty = taker.FilledQty.Add(tradeQty)
			level.TotalQty = level.TotalQty.Sub(tradeQty)
			tradeAmount = tradeAmount.Add(tradeQty)

			// Maker 成交完，移除
			if maker.Quantity.IsZero() {
				level.Orders.Remove(e)
				delete(ob.OrderIndex, maker.ID)
				maker.Status = domain.Filled
			} else {
				maker.Status = domain.PartiallyFilled
			}

			if taker.Quantity.IsZero() {
				break
			}
		}

		// 如果该档位空了，移除档位
		if level.Orders.Len() == 0 {
			delete(ob.Asks, bestAskPrice.String())
			// 移除价格 Slice (性能较低，生产建议用 RingBuffer 或 Heap)
			ob.AskPrices = append(ob.AskPrices[:i], ob.AskPrices[i+1:]...)
			// i 不变，继续检查下一个 bestAsk
		} else {
			i++
		}

		if taker.Quantity.IsZero() {
			taker.Status = domain.Filled
			break
		}
	}
	return trades
}

// matchAsk 撮合卖单：去吃 Bid (High -> Low)
func (ob *InMemoryOrderBook) matchAsk(taker *domain.Order) []*domain.Trade {
	var trades []*domain.Trade

	// 遍历买单价格 (降序)
	for i := 0; i < len(ob.BidPrices); {
		bestBidPrice := ob.BidPrices[i]

		// 价格不满足（卖价比买一高），停止撮合
		if taker.Type == domain.Limit && taker.Price.GreaterThan(bestBidPrice) {
			break
		}

		level := ob.Bids[bestBidPrice.String()]

		var next *list.Element
		for e := level.Orders.Front(); e != nil; e = next {
			next = e.Next()
			maker := e.Value.(*domain.Order)

			tradeQty := decimal.Min(taker.Quantity, maker.Quantity)

			trades = append(trades, &domain.Trade{
				TradeID:      fmt.Sprintf("T-%d-%d-%d", maker.ID, taker.ID, taker.Timestamp),
				BuyOrderID:   fmt.Sprintf("%d", maker.ID),
				SellOrderID:  fmt.Sprintf("%d", taker.ID),
				Symbol:       ob.Symbol,
				Price:        maker.Price.InexactFloat64(),
				Quantity:     tradeQty.InexactFloat64(),
				BuyerID:      maker.ID, // 买方是 Maker
				SellerID:     taker.ID, // 卖方是 Taker
				TakerOrder:   taker.ID,
				MakerOrders:  []uint64{maker.ID},
				Timestamp:    time.Unix(0, taker.Timestamp),
				MakerOrderID: maker.ID,
				TakerOrderID: taker.ID,
				Qty:          tradeQty.IntPart(),
				TradeTime:    taker.Timestamp,
			})

			maker.Quantity = maker.Quantity.Sub(tradeQty)
			maker.FilledQty = maker.FilledQty.Add(tradeQty)
			taker.Quantity = taker.Quantity.Sub(tradeQty)
			taker.FilledQty = taker.FilledQty.Add(tradeQty)
			level.TotalQty = level.TotalQty.Sub(tradeQty)

			if maker.Quantity.IsZero() {
				level.Orders.Remove(e)
				delete(ob.OrderIndex, maker.ID)
				maker.Status = domain.Filled
			} else {
				maker.Status = domain.PartiallyFilled
			}

			if taker.Quantity.IsZero() {
				break
			}
		}

		if level.Orders.Len() == 0 {
			delete(ob.Bids, bestBidPrice.String())
			ob.BidPrices = append(ob.BidPrices[:i], ob.BidPrices[i+1:]...)
		} else {
			i++
		}

		if taker.Quantity.IsZero() {
			taker.Status = domain.Filled
			break
		}
	}
	return trades
}

func (ob *InMemoryOrderBook) addRestOrder(order *domain.Order) {
	priceStr := order.Price.String()
	var level *OrderLevel
	var ok bool

	if order.Side == domain.Buy {
		if level, ok = ob.Bids[priceStr]; !ok {
			level = NewOrderLevel(order.Price)
			ob.Bids[priceStr] = level
			// 插入排序 (降序)
			insertIdx := sort.Search(len(ob.BidPrices), func(i int) bool {
				return ob.BidPrices[i].LessThanOrEqual(order.Price)
			})
			// 检查重复，只在不相等时插入
			if insertIdx < len(ob.BidPrices) && ob.BidPrices[insertIdx].Equal(order.Price) {
				// exists
			} else {
				// 你的 sort.Search 是给升序用的吗？
				// 我们需要手动维护降序数组
				// 简单点：append 然后 sort
				ob.BidPrices = append(ob.BidPrices, order.Price)
				sort.Slice(ob.BidPrices, func(i, j int) bool {
					return ob.BidPrices[i].GreaterThan(ob.BidPrices[j]) // Desc
				})
			}
		}
	} else {
		if level, ok = ob.Asks[priceStr]; !ok {
			level = NewOrderLevel(order.Price)
			ob.Asks[priceStr] = level
			ob.AskPrices = append(ob.AskPrices, order.Price)
			sort.Slice(ob.AskPrices, func(i, j int) bool {
				return ob.AskPrices[i].LessThan(ob.AskPrices[j]) // Asc
			})
		}
	}

	elem := level.Orders.PushBack(order)
	level.TotalQty = level.TotalQty.Add(order.Quantity)
	ob.OrderIndex[order.ID] = elem
}

// CancelOrder 取消订单
func (ob *InMemoryOrderBook) CancelOrder(orderID uint64) (*domain.Order, error) {
	elem, ok := ob.OrderIndex[orderID]
	if !ok {
		return nil, errors.New("order not found")
	}

	order := elem.Value.(*domain.Order)
	priceStr := order.Price.String()

	var level *OrderLevel
	if order.Side == domain.Buy {
		level = ob.Bids[priceStr]
	} else {
		level = ob.Asks[priceStr]
	}

	if level != nil {
		level.Orders.Remove(elem)
		level.TotalQty = level.TotalQty.Sub(order.Quantity)
		if level.Orders.Len() == 0 {
			// 清理空 Level (可选优化)
			if order.Side == domain.Buy {
				delete(ob.Bids, priceStr)
				// 清理 prices array 较慢，暂略
			} else {
				delete(ob.Asks, priceStr)
			}
		}
	}

	delete(ob.OrderIndex, orderID)
	order.Status = domain.Canceled
	return order, nil
}

// GetDepth 获取 L2 深度
func (ob *InMemoryOrderBook) GetDepth(limit int) (bids, asks []domain.PriceLevel) {
	count := 0
	for _, p := range ob.BidPrices {
		if l, ok := ob.Bids[p.String()]; ok && l.Orders.Len() > 0 {
			bids = append(bids, domain.PriceLevel{Price: p, Quantity: l.TotalQty})
			count++
			if count >= limit {
				break
			}
		}
	}

	count = 0
	for _, p := range ob.AskPrices {
		if l, ok := ob.Asks[p.String()]; ok && l.Orders.Len() > 0 {
			asks = append(asks, domain.PriceLevel{Price: p, Quantity: l.TotalQty})
			count++
			if count >= limit {
				break
			}
		}
	}
	return
}
