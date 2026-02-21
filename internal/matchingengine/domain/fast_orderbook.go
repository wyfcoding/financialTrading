// 变更说明：
// 面向 CPU L1/L2 Cache 优化的极速订单簿 (L2 OrderBook)。
// 核心优化点：
// 1. 杜用 "decimal.Decimal" (底层使用了 big.Int 会在堆上分配内存)，将所有价格和数量转化为在网关层放大 10000 倍的 int64。
// 2. 杜绝使用 container/list 双向链表，改用连续内存切片 (Slice) 或自平衡数组，极大提升缓存命中率 (Cache Hit)。
// 3. 将 Order 的创建变为对象池复用，订单薄里只存数据的指针或扁平结构。
package domain

import (
	"fmt"
	"sort"
	"time"
)

// OrderSide 买卖方向
type OrderSide int8

const (
	Buy  OrderSide = 1
	Sell OrderSide = 2
)

// FastOrder 零分配订单对象。在 RingBuffer 中它被预先分配好。
type FastOrder struct {
	OrderID   uint64
	AccountID uint64
	Price     int64 // 放大 10000 倍后的定点整数
	Qty       int64 // 放大 10000 倍后的定点整数
	Time      int64 // 纳秒时间戳，用于时间优先排序
}

// OrderLevel 价格档位。
// 完全摒弃指针链表，使用基于容量扩容的连续数组 []FastOrder 以迎合 CPU 缓存预取 (Prefetching)。
type OrderLevel struct {
	Price  int64
	Volume int64
	Orders []FastOrder
}

// FastOrderBook 极速内存订单簿。
type FastOrderBook struct {
	Symbol string
	Bids   []*OrderLevel // 买盘 (降序排序)
	Asks   []*OrderLevel // 卖盘 (升序排序)
}

func NewFastOrderBook(symbol string) *FastOrderBook {
	return &FastOrderBook{
		Symbol: symbol,
		// 预分配常用档位，减少运行时的切片扩容开销
		Bids: make([]*OrderLevel, 0, 1000),
		Asks: make([]*OrderLevel, 0, 1000),
	}
}

// Trade 撮合成功的成交记录对象
type Trade struct {
	ID          uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
	TradeID     string
	BuyOrderID  string
	SellOrderID string
	Symbol      string
	Price       float64
	Quantity    float64
	Timestamp   time.Time
	BuyerID     uint64
	SellerID    uint64
	TakerOrder  uint64
	MakerOrders []uint64

	MakerOrderID uint64
	TakerOrderID uint64
	Qty          int64
	TradeTime    int64
}

// Process 限价单撮合处理，返回产生的 Transaction(Trades) 列表。
func (ob *FastOrderBook) Process(order FastOrder, side OrderSide) []Trade {
	var trades []Trade
	remainingQty := order.Qty

	if side == Buy {
		// 买单，吃掉 Asks (卖盘)
		for len(ob.Asks) > 0 && remainingQty > 0 {
			bestAsk := ob.Asks[0]
			if order.Price < bestAsk.Price {
				break // 买价低于卖一价，无法成交
			}

			// 在当前价格档位挨个吃掉排队的挂单 (Price-Time Priority)
			trades, remainingQty = ob.matchLevel(bestAsk, order, remainingQty, false, trades)

			if len(bestAsk.Orders) == 0 {
				// 卖一档位被全部吃光，移除该档位 (因为是指针数组，直接切片偏移，O(1)成本)
				ob.Asks = ob.Asks[1:]
			}
		}
		// 如果还有剩余数量，挂入 Bids (买盘)
		if remainingQty > 0 {
			order.Qty = remainingQty
			ob.addOrderToLevel(&ob.Bids, order, true)
		}
	} else {
		// 卖单，吃掉 Bids (买盘)
		for len(ob.Bids) > 0 && remainingQty > 0 {
			bestBid := ob.Bids[0]
			if order.Price > bestBid.Price {
				break // 卖价高于买一价，无法成交
			}

			trades, remainingQty = ob.matchLevel(bestBid, order, remainingQty, true, trades)

			if len(bestBid.Orders) == 0 {
				// 买一档位被全部吃光
				ob.Bids = ob.Bids[1:]
			}
		}
		if remainingQty > 0 {
			order.Qty = remainingQty
			ob.addOrderToLevel(&ob.Asks, order, false)
		}
	}

	return trades
}

// matchLevel 遍历并撮合指定价格档位上的所有订单
func (ob *FastOrderBook) matchLevel(level *OrderLevel, taker FastOrder, remainingQty int64, isTakerSell bool, trades []Trade) ([]Trade, int64) {
	for i := 0; i < len(level.Orders) && remainingQty > 0; i++ {
		maker := &level.Orders[i]
		tradeQty := remainingQty
		if maker.Qty < tradeQty {
			tradeQty = maker.Qty
		}

		trades = append(trades, Trade{
			TradeID:      fmt.Sprintf("T-%d-%d-%d", maker.OrderID, taker.OrderID, taker.Time),
			BuyOrderID:   fmt.Sprintf("%d", taker.OrderID),
			SellOrderID:  fmt.Sprintf("%d", maker.OrderID),
			Symbol:       ob.Symbol,
			Price:        float64(level.Price),
			Quantity:     float64(tradeQty),
			Timestamp:    time.Unix(0, taker.Time),
			BuyerID:      taker.AccountID,
			SellerID:     maker.AccountID,
			TakerOrder:   taker.OrderID,
			MakerOrders:  []uint64{maker.OrderID},
			MakerOrderID: maker.OrderID,
			TakerOrderID: taker.OrderID,
			Qty:          tradeQty,
			TradeTime:    taker.Time,
		})

		maker.Qty -= tradeQty
		level.Volume -= tradeQty
		remainingQty -= tradeQty
	}

	// 移除该档位中已经完全成交的订单
	// 通过寻找第一个未成交的 index，进行高效切片原地截断
	firstAlive := 0
	for ; firstAlive < len(level.Orders); firstAlive++ {
		if level.Orders[firstAlive].Qty > 0 {
			break
		}
	}
	level.Orders = level.Orders[firstAlive:]

	return trades, remainingQty
}

// addOrderToLevel 挂单插入数组 (使用二分查找 O(log N) 定位价格，再插入)
func (ob *FastOrderBook) addOrderToLevel(levels *[]*OrderLevel, order FastOrder, isBid bool) {
	lvls := *levels

	// Binary Search 寻找对应的 Price Level
	idx := sort.Search(len(lvls), func(i int) bool {
		if isBid {
			return lvls[i].Price <= order.Price // 降序
		}
		return lvls[i].Price >= order.Price // 升序
	})

	if idx < len(lvls) && lvls[idx].Price == order.Price {
		// 档位已存在，Append 订单
		lvls[idx].Orders = append(lvls[idx].Orders, order)
		lvls[idx].Volume += order.Qty
	} else {
		// 新价格档位，执行插入 (创建定长 Slice 减少扩容)
		newLevel := &OrderLevel{
			Price:  order.Price,
			Volume: order.Qty,
			Orders: make([]FastOrder, 0, 8),
		}
		newLevel.Orders = append(newLevel.Orders, order)

		*levels = append(lvls[:idx], append([]*OrderLevel{newLevel}, lvls[idx:]...)...)
	}
}

// PrintBook 仅供 Debug 使用
func (ob *FastOrderBook) PrintBook() {
	fmt.Printf("--- ORDER BOOK: %s ---\n", ob.Symbol)
	for i := len(ob.Asks) - 1; i >= 0; i-- {
		fmt.Printf("ASK: %d | VOL: %d\n", ob.Asks[i].Price, ob.Asks[i].Volume)
	}
	fmt.Println("-----------------------")
	for _, bid := range ob.Bids {
		fmt.Printf("BID: %d | VOL: %d\n", bid.Price, bid.Volume)
	}
}
