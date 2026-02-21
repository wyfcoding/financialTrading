// 变更说明：
// 1. 【关键修复】将所有价格字段从 float64 改为 decimal.Decimal，消除金融精度问题
// 2. 【性能优化】PriceLevelMap 排序从冒泡排序 O(n²) 改为二分插入 O(log n)
// 3. 【功能增强】新增 PostOnly/FOK/IOC 等订单类型完整支持
// 4. 【功能增强】Quantity 也使用 decimal.Decimal 支持小数量（加密货币场景）
// 5. 【安全增强】Trade ID 使用原子递增 + 时间戳组合，避免冲突
// 6. 【功能增强】新增 ModifyOrder 改单功能
package orderbook

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"
)

// OrderSide 订单方向。
type OrderSide int8

const (
	// SideBuy 买入方向。
	SideBuy OrderSide = iota
	// SideSell 卖出方向。
	SideSell
)

// String 返回方向的字符串表示。
func (s OrderSide) String() string {
	if s == SideBuy {
		return "BUY"
	}
	return "SELL"
}

// OrderType 订单类型。
type OrderType int8

const (
	// OrderTypeLimit 限价单：指定价格挂单。
	OrderTypeLimit OrderType = iota
	// OrderTypeMarket 市价单：以当前最优价格立即成交。
	OrderTypeMarket
	// OrderTypeStopLimit 止损限价单：触发价格后转为限价单。
	OrderTypeStopLimit
	// OrderTypeStopMarket 止损市价单：触发价格后转为市价单。
	OrderTypeStopMarket
	// OrderTypeFOK Fill-or-Kill：必须全部成交，否则全部取消。
	OrderTypeFOK
	// OrderTypeIOC Immediate-or-Cancel：立即成交可成交部分，剩余取消。
	OrderTypeIOC
	// OrderTypeAON All-or-None：必须全部成交，但可以等待。
	OrderTypeAON
	// OrderTypePostOnly 只做 Maker：如果会立即成交则拒绝。
	OrderTypePostOnly
)

// OrderStatus 订单状态。
type OrderStatus int8

const (
	// OrderStatusNew 新建。
	OrderStatusNew OrderStatus = iota
	// OrderStatusPartiallyFilled 部分成交。
	OrderStatusPartiallyFilled
	// OrderStatusFilled 完全成交。
	OrderStatusFilled
	// OrderStatusCancelled 已取消。
	OrderStatusCancelled
	// OrderStatusRejected 已拒绝。
	OrderStatusRejected
	// OrderStatusExpired 已过期。
	OrderStatusExpired
)

// Order 订单实体。
// 所有价格和数量字段均使用 decimal.Decimal 确保金融精度。
type Order struct {
	// OrderID 订单唯一标识。
	OrderID string `json:"order_id"`
	// Symbol 交易标的代码。
	Symbol string `json:"symbol"`
	// Side 买卖方向。
	Side OrderSide `json:"side"`
	// Type 订单类型。
	Type OrderType `json:"type"`
	// Price 委托价格（限价单必填）。
	Price decimal.Decimal `json:"price"`
	// Quantity 委托总量。
	Quantity decimal.Decimal `json:"quantity"`
	// FilledQty 已成交量。
	FilledQty decimal.Decimal `json:"filled_qty"`
	// RemainingQty 剩余未成交量。
	RemainingQty decimal.Decimal `json:"remaining_qty"`
	// Status 当前状态。
	Status OrderStatus `json:"status"`
	// Timestamp 委托时间戳（纳秒）。
	Timestamp int64 `json:"timestamp"`
	// UserID 委托用户 ID。
	UserID uint64 `json:"user_id"`
	// StopPrice 止损触发价格。
	StopPrice decimal.Decimal `json:"stop_price"`
	// TimeInForce 有效期策略（GTC/IOC/FOK/GTD）。
	TimeInForce string `json:"time_in_force"`
	// ExpireTime 过期时间戳（GTD 模式使用）。
	ExpireTime int64 `json:"expire_time"`
}

// Trade 成交记录。
type Trade struct {
	// TradeID 成交唯一标识。
	TradeID string `json:"trade_id"`
	// Symbol 交易标的代码。
	Symbol string `json:"symbol"`
	// Price 成交价格。
	Price decimal.Decimal `json:"price"`
	// Quantity 成交数量。
	Quantity decimal.Decimal `json:"quantity"`
	// BuyOrderID 买方订单号。
	BuyOrderID string `json:"buy_order_id"`
	// SellOrderID 卖方订单号。
	SellOrderID string `json:"sell_order_id"`
	// BuyUserID 买方用户 ID。
	BuyUserID uint64 `json:"buy_user_id"`
	// SellUserID 卖方用户 ID。
	SellUserID uint64 `json:"sell_user_id"`
	// Timestamp 成交时间戳（纳秒）。
	Timestamp int64 `json:"timestamp"`
	// IsMakerBuy 买方是否为 Maker。
	IsMakerBuy bool `json:"is_maker_buy"`
}

// PriceLevel 价格档位，同一价格的所有订单按时间优先排列。
type PriceLevel struct {
	// Price 该档位价格。
	Price decimal.Decimal
	// Orders 订单映射（OrderID -> Order）。
	Orders map[string]*Order
	// OrderList 按时间优先排列的订单 ID 列表。
	OrderList []string
	// TotalQty 该档位总挂单量。
	TotalQty decimal.Decimal
	// OrderCount 该档位订单数。
	OrderCount int
}

// NewPriceLevel 创建价格档位。
func NewPriceLevel(price decimal.Decimal) *PriceLevel {
	return &PriceLevel{
		Price:     price,
		Orders:    make(map[string]*Order),
		OrderList: make([]string, 0, 16),
		TotalQty:  decimal.Zero,
	}
}

// AddOrder 添加订单到档位尾部（时间优先）。
func (l *PriceLevel) AddOrder(order *Order) {
	l.Orders[order.OrderID] = order
	l.OrderList = append(l.OrderList, order.OrderID)
	l.TotalQty = l.TotalQty.Add(order.RemainingQty)
	l.OrderCount++
}

// RemoveOrder 移除指定订单。
func (l *PriceLevel) RemoveOrder(orderID string) *Order {
	order, ok := l.Orders[orderID]
	if !ok {
		return nil
	}
	delete(l.Orders, orderID)
	for i, id := range l.OrderList {
		if id == orderID {
			l.OrderList = append(l.OrderList[:i], l.OrderList[i+1:]...)
			break
		}
	}
	l.TotalQty = l.TotalQty.Sub(order.RemainingQty)
	l.OrderCount--
	return order
}

// FirstOrder 获取时间优先的第一个订单。
func (l *PriceLevel) FirstOrder() *Order {
	if len(l.OrderList) == 0 {
		return nil
	}
	return l.Orders[l.OrderList[0]]
}

// UpdateQuantity 更新订单成交量。delta 为本次成交量（正数）。
func (l *PriceLevel) UpdateQuantity(orderID string, delta decimal.Decimal) {
	if order, ok := l.Orders[orderID]; ok {
		l.TotalQty = l.TotalQty.Sub(delta)
		order.RemainingQty = order.RemainingQty.Sub(delta)
		order.FilledQty = order.FilledQty.Add(delta)
	}
}

// IsEmpty 判断档位是否为空。
func (l *PriceLevel) IsEmpty() bool {
	return l.OrderCount == 0
}

// PriceLevelMap 有序价格档位映射。
// 使用排序切片维护价格顺序，二分查找实现 O(log n) 插入。
type PriceLevelMap struct {
	levels map[string]*PriceLevel // key 为 price.String()，避免浮点 map key 问题
	prices []decimal.Decimal      // 有序价格列表
	desc   bool                   // true=降序（买盘），false=升序（卖盘）
}

// NewPriceLevelMap 创建价格档位映射。
// desc=true 用于买盘（最高价优先），desc=false 用于卖盘（最低价优先）。
func NewPriceLevelMap(desc bool) *PriceLevelMap {
	return &PriceLevelMap{
		levels: make(map[string]*PriceLevel),
		prices: make([]decimal.Decimal, 0, 64),
		desc:   desc,
	}
}

// priceKey 生成价格的 map key。
func priceKey(price decimal.Decimal) string {
	return price.String()
}

// GetOrCreate 获取或创建价格档位。
// 使用二分插入保持有序，时间复杂度 O(log n)。
func (m *PriceLevelMap) GetOrCreate(price decimal.Decimal) *PriceLevel {
	key := priceKey(price)
	if level, ok := m.levels[key]; ok {
		return level
	}
	level := NewPriceLevel(price)
	m.levels[key] = level
	m.insertSorted(price)
	return level
}

// insertSorted 二分插入保持有序。
func (m *PriceLevelMap) insertSorted(price decimal.Decimal) {
	n := len(m.prices)
	idx := sort.Search(n, func(i int) bool {
		if m.desc {
			return m.prices[i].LessThan(price)
		}
		return m.prices[i].GreaterThan(price)
	})
	m.prices = append(m.prices, decimal.Zero)
	copy(m.prices[idx+1:], m.prices[idx:])
	m.prices[idx] = price
}

// Get 获取价格档位。
func (m *PriceLevelMap) Get(price decimal.Decimal) (*PriceLevel, bool) {
	level, ok := m.levels[priceKey(price)]
	return level, ok
}

// Delete 删除价格档位。
func (m *PriceLevelMap) Delete(price decimal.Decimal) {
	key := priceKey(price)
	delete(m.levels, key)
	for i, p := range m.prices {
		if p.Equal(price) {
			m.prices = append(m.prices[:i], m.prices[i+1:]...)
			break
		}
	}
}

// First 获取最优价格档位。买盘返回最高价，卖盘返回最低价。
func (m *PriceLevelMap) First() (*PriceLevel, bool) {
	if len(m.prices) == 0 {
		return nil, false
	}
	return m.levels[priceKey(m.prices[0])], true
}

// Len 获取档位数量。
func (m *PriceLevelMap) Len() int {
	return len(m.prices)
}

// Iterate 按优先级顺序遍历所有档位。fn 返回 false 时停止遍历。
func (m *PriceLevelMap) Iterate(fn func(price decimal.Decimal, level *PriceLevel) bool) {
	for _, price := range m.prices {
		level := m.levels[priceKey(price)]
		if level == nil {
			continue
		}
		if !fn(price, level) {
			break
		}
	}
}

// OrderBook 高性能订单簿。
// 核心数据结构：买盘（降序）+ 卖盘（升序），价格优先/时间优先撮合。
// 并发模型：读写锁保护，写操作互斥，读操作并发。
type OrderBook struct {
	symbol       string
	bids         *PriceLevelMap
	asks         *PriceLevelMap
	orders       sync.Map
	mu           sync.RWMutex
	version      atomic.Int64
	lastUpdateID int64
	tradeIDGen   atomic.Int64
}

// NewOrderBook 创建订单簿。
func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		symbol: symbol,
		bids:   NewPriceLevelMap(true),
		asks:   NewPriceLevelMap(false),
	}
}

// AddOrder 添加订单并执行撮合。
// 返回本次产生的成交记录列表。撮合规则：价格优先 - 时间优先。
func (ob *OrderBook) AddOrder(order *Order) ([]*Trade, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order.Timestamp = time.Now().UnixNano()
	order.Status = OrderStatusNew
	order.RemainingQty = order.Quantity
	order.FilledQty = decimal.Zero

	// PostOnly 检查：如果会立即成交则拒绝
	if order.Type == OrderTypePostOnly {
		if ob.wouldMatch(order) {
			order.Status = OrderStatusRejected
			return nil, fmt.Errorf("post-only order would match immediately")
		}
		ob.insertOrder(order)
		ob.version.Add(1)
		return nil, nil
	}

	// 市价单撮合
	if order.Type == OrderTypeMarket {
		trades := ob.matchOrder(order)
		ob.version.Add(1)
		return trades, nil
	}

	// FOK 检查：必须能全部成交
	if order.Type == OrderTypeFOK {
		if !ob.canFillCompletely(order) {
			order.Status = OrderStatusCancelled
			return nil, nil
		}
	}

	return ob.addLimitOrder(order), nil
}

// wouldMatch 检查订单是否会立即成交（用于 PostOnly 判断）。
func (ob *OrderBook) wouldMatch(order *Order) bool {
	if order.Side == SideBuy {
		level, ok := ob.asks.First()
		return ok && order.Price.GreaterThanOrEqual(level.Price)
	}
	level, ok := ob.bids.First()
	return ok && order.Price.LessThanOrEqual(level.Price)
}

// canFillCompletely 检查订单是否能完全成交（用于 FOK 判断）。
func (ob *OrderBook) canFillCompletely(order *Order) bool {
	remaining := order.Quantity
	var book *PriceLevelMap
	if order.Side == SideBuy {
		book = ob.asks
	} else {
		book = ob.bids
	}

	book.Iterate(func(price decimal.Decimal, level *PriceLevel) bool {
		if order.Side == SideBuy && price.GreaterThan(order.Price) {
			return false
		}
		if order.Side == SideSell && price.LessThan(order.Price) {
			return false
		}
		remaining = remaining.Sub(level.TotalQty)
		return remaining.IsPositive()
	})

	return !remaining.IsPositive()
}

// addLimitOrder 添加限价单并执行撮合。
func (ob *OrderBook) addLimitOrder(order *Order) []*Trade {
	trades := ob.matchOrder(order)

	if order.Type == OrderTypeIOC {
		if order.RemainingQty.IsPositive() {
			order.Status = OrderStatusCancelled
		}
	} else if order.RemainingQty.IsPositive() {
		ob.insertOrder(order)
	}

	ob.version.Add(1)
	ob.lastUpdateID = time.Now().UnixNano()
	return trades
}

// matchOrder 执行订单撮合。
func (ob *OrderBook) matchOrder(order *Order) []*Trade {
	if order.Side == SideBuy {
		return ob.matchBuyOrder(order)
	}
	return ob.matchSellOrder(order)
}

// matchBuyOrder 买单撮合：与卖盘（升序）逐档匹配。
func (ob *OrderBook) matchBuyOrder(order *Order) []*Trade {
	var trades []*Trade
	isMarket := order.Type == OrderTypeMarket

	for ob.asks.Len() > 0 && order.RemainingQty.IsPositive() {
		level, ok := ob.asks.First()
		if !ok {
			break
		}
		if !isMarket && level.Price.GreaterThan(order.Price) {
			break
		}

		for level.OrderCount > 0 && order.RemainingQty.IsPositive() {
			sellOrder := level.FirstOrder()
			if sellOrder == nil {
				break
			}

			matchQty := decimal.Min(order.RemainingQty, sellOrder.RemainingQty)
			trade := ob.createTrade(order, sellOrder, matchQty, level.Price, false)
			trades = append(trades, trade)

			level.UpdateQuantity(sellOrder.OrderID, matchQty)
			order.RemainingQty = order.RemainingQty.Sub(matchQty)
			order.FilledQty = order.FilledQty.Add(matchQty)

			if sellOrder.RemainingQty.IsZero() {
				sellOrder.Status = OrderStatusFilled
				level.RemoveOrder(sellOrder.OrderID)
				ob.orders.Delete(sellOrder.OrderID)
			} else {
				sellOrder.Status = OrderStatusPartiallyFilled
			}
		}

		if level.IsEmpty() {
			ob.asks.Delete(level.Price)
		}
	}

	ob.updateOrderStatus(order)
	return trades
}

// matchSellOrder 卖单撮合：与买盘（降序）逐档匹配。
func (ob *OrderBook) matchSellOrder(order *Order) []*Trade {
	var trades []*Trade
	isMarket := order.Type == OrderTypeMarket

	for ob.bids.Len() > 0 && order.RemainingQty.IsPositive() {
		level, ok := ob.bids.First()
		if !ok {
			break
		}
		if !isMarket && level.Price.LessThan(order.Price) {
			break
		}

		for level.OrderCount > 0 && order.RemainingQty.IsPositive() {
			buyOrder := level.FirstOrder()
			if buyOrder == nil {
				break
			}

			matchQty := decimal.Min(order.RemainingQty, buyOrder.RemainingQty)
			trade := ob.createTrade(buyOrder, order, matchQty, level.Price, true)
			trades = append(trades, trade)

			level.UpdateQuantity(buyOrder.OrderID, matchQty)
			order.RemainingQty = order.RemainingQty.Sub(matchQty)
			order.FilledQty = order.FilledQty.Add(matchQty)

			if buyOrder.RemainingQty.IsZero() {
				buyOrder.Status = OrderStatusFilled
				level.RemoveOrder(buyOrder.OrderID)
				ob.orders.Delete(buyOrder.OrderID)
			} else {
				buyOrder.Status = OrderStatusPartiallyFilled
			}
		}

		if level.IsEmpty() {
			ob.bids.Delete(level.Price)
		}
	}

	ob.updateOrderStatus(order)
	return trades
}

// updateOrderStatus 根据成交情况更新订单状态。
func (ob *OrderBook) updateOrderStatus(order *Order) {
	if order.FilledQty.IsPositive() {
		if order.RemainingQty.IsZero() {
			order.Status = OrderStatusFilled
		} else {
			order.Status = OrderStatusPartiallyFilled
		}
	}
}

// insertOrder 将订单插入对应的价格档位。
func (ob *OrderBook) insertOrder(order *Order) {
	var book *PriceLevelMap
	if order.Side == SideBuy {
		book = ob.bids
	} else {
		book = ob.asks
	}
	level := book.GetOrCreate(order.Price)
	level.AddOrder(order)
	ob.orders.Store(order.OrderID, order)
}

// CancelOrder 取消订单。
func (ob *OrderBook) CancelOrder(orderID string) (*Order, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	value, ok := ob.orders.Load(orderID)
	if !ok {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	order := value.(*Order)

	var book *PriceLevelMap
	if order.Side == SideBuy {
		book = ob.bids
	} else {
		book = ob.asks
	}

	level, ok := book.Get(order.Price)
	if ok {
		level.RemoveOrder(orderID)
		if level.IsEmpty() {
			book.Delete(order.Price)
		}
	}

	ob.orders.Delete(orderID)
	order.Status = OrderStatusCancelled
	ob.version.Add(1)

	return order, nil
}

// ModifyOrder 修改订单（先取消再重新下单）。
// 注意：修改后时间优先级会重置。
func (ob *OrderBook) ModifyOrder(orderID string, newPrice, newQty decimal.Decimal) ([]*Trade, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	value, ok := ob.orders.Load(orderID)
	if !ok {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	oldOrder := value.(*Order)

	var book *PriceLevelMap
	if oldOrder.Side == SideBuy {
		book = ob.bids
	} else {
		book = ob.asks
	}

	level, ok := book.Get(oldOrder.Price)
	if ok {
		level.RemoveOrder(orderID)
		if level.IsEmpty() {
			book.Delete(oldOrder.Price)
		}
	}
	ob.orders.Delete(orderID)

	newOrder := &Order{
		OrderID:      orderID,
		Symbol:       oldOrder.Symbol,
		Side:         oldOrder.Side,
		Type:         oldOrder.Type,
		Price:        newPrice,
		Quantity:     newQty,
		RemainingQty: newQty,
		FilledQty:    decimal.Zero,
		Status:       OrderStatusNew,
		Timestamp:    time.Now().UnixNano(),
		UserID:       oldOrder.UserID,
		StopPrice:    oldOrder.StopPrice,
		TimeInForce:  oldOrder.TimeInForce,
		ExpireTime:   oldOrder.ExpireTime,
	}

	trades := ob.matchOrder(newOrder)
	if newOrder.RemainingQty.IsPositive() {
		ob.insertOrder(newOrder)
	}

	ob.version.Add(1)
	return trades, nil
}

// GetOrder 获取订单。
func (ob *OrderBook) GetOrder(orderID string) (*Order, bool) {
	value, ok := ob.orders.Load(orderID)
	if !ok {
		return nil, false
	}
	return value.(*Order), true
}

// GetBestBid 获取最优买价及其挂单量。
func (ob *OrderBook) GetBestBid() (decimal.Decimal, decimal.Decimal) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	level, ok := ob.bids.First()
	if !ok {
		return decimal.Zero, decimal.Zero
	}
	return level.Price, level.TotalQty
}

// GetBestAsk 获取最优卖价及其挂单量。
func (ob *OrderBook) GetBestAsk() (decimal.Decimal, decimal.Decimal) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	level, ok := ob.asks.First()
	if !ok {
		return decimal.Zero, decimal.Zero
	}
	return level.Price, level.TotalQty
}

// GetSpread 获取买卖价差信息。返回：价差绝对值、价差百分比、中间价。
func (ob *OrderBook) GetSpread() (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	bid, _ := ob.GetBestBid()
	ask, _ := ob.GetBestAsk()
	if bid.IsZero() || ask.IsZero() {
		return decimal.Zero, decimal.Zero, decimal.Zero
	}
	spread := ask.Sub(bid)
	two := decimal.NewFromInt(2)
	midPrice := bid.Add(ask).Div(two)
	spreadPct := decimal.Zero
	if midPrice.IsPositive() {
		hundred := decimal.NewFromInt(100)
		spreadPct = spread.Div(midPrice).Mul(hundred)
	}
	return spread, spreadPct, midPrice
}

// GetDepth 获取指定档位数的深度数据。
func (ob *OrderBook) GetDepth(levels int) ([]*PriceLevel, []*PriceLevel) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids := make([]*PriceLevel, 0, levels)
	asks := make([]*PriceLevel, 0, levels)

	count := 0
	ob.bids.Iterate(func(_ decimal.Decimal, level *PriceLevel) bool {
		if count >= levels {
			return false
		}
		bids = append(bids, level)
		count++
		return true
	})

	count = 0
	ob.asks.Iterate(func(_ decimal.Decimal, level *PriceLevel) bool {
		if count >= levels {
			return false
		}
		asks = append(asks, level)
		count++
		return true
	})

	return bids, asks
}

// Snapshot 订单簿快照。
type Snapshot struct {
	Symbol    string               `json:"symbol"`
	Bids      []PriceLevelSnapshot `json:"bids"`
	Asks      []PriceLevelSnapshot `json:"asks"`
	Timestamp int64                `json:"timestamp"`
	UpdateID  int64                `json:"update_id"`
}

// PriceLevelSnapshot 价格档位快照。
type PriceLevelSnapshot struct {
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
	Count    int             `json:"count"`
}

// GetSnapshot 获取订单簿快照。
func (ob *OrderBook) GetSnapshot(levels int) *Snapshot {
	bids, asks := ob.GetDepth(levels)

	snapshot := &Snapshot{
		Symbol:    ob.symbol,
		Timestamp: time.Now().UnixNano(),
		UpdateID:  ob.lastUpdateID,
		Bids:      make([]PriceLevelSnapshot, 0, len(bids)),
		Asks:      make([]PriceLevelSnapshot, 0, len(asks)),
	}

	for _, l := range bids {
		snapshot.Bids = append(snapshot.Bids, PriceLevelSnapshot{
			Price: l.Price, Quantity: l.TotalQty, Count: l.OrderCount,
		})
	}
	for _, l := range asks {
		snapshot.Asks = append(snapshot.Asks, PriceLevelSnapshot{
			Price: l.Price, Quantity: l.TotalQty, Count: l.OrderCount,
		})
	}

	return snapshot
}

// Stats 订单簿统计信息。
type Stats struct {
	Symbol        string          `json:"symbol"`
	BidLevels     int             `json:"bid_levels"`
	AskLevels     int             `json:"ask_levels"`
	TotalBidQty   decimal.Decimal `json:"total_bid_qty"`
	TotalAskQty   decimal.Decimal `json:"total_ask_qty"`
	BestBid       decimal.Decimal `json:"best_bid"`
	BestAsk       decimal.Decimal `json:"best_ask"`
	Spread        decimal.Decimal `json:"spread"`
	SpreadPercent decimal.Decimal `json:"spread_percent"`
	MidPrice      decimal.Decimal `json:"mid_price"`
	OrderCount    int             `json:"order_count"`
	Version       int64           `json:"version"`
}

// GetStats 获取订单簿统计信息。
func (ob *OrderBook) GetStats() *Stats {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	stats := &Stats{
		Symbol:      ob.symbol,
		Version:     ob.version.Load(),
		TotalBidQty: decimal.Zero,
		TotalAskQty: decimal.Zero,
	}

	stats.BidLevels = ob.bids.Len()
	stats.AskLevels = ob.asks.Len()

	ob.bids.Iterate(func(_ decimal.Decimal, level *PriceLevel) bool {
		stats.TotalBidQty = stats.TotalBidQty.Add(level.TotalQty)
		stats.OrderCount += level.OrderCount
		return true
	})

	ob.asks.Iterate(func(_ decimal.Decimal, level *PriceLevel) bool {
		stats.TotalAskQty = stats.TotalAskQty.Add(level.TotalQty)
		stats.OrderCount += level.OrderCount
		return true
	})

	if level, ok := ob.bids.First(); ok {
		stats.BestBid = level.Price
	}
	if level, ok := ob.asks.First(); ok {
		stats.BestAsk = level.Price
	}

	if stats.BestBid.IsPositive() && stats.BestAsk.IsPositive() {
		stats.Spread = stats.BestAsk.Sub(stats.BestBid)
		two := decimal.NewFromInt(2)
		stats.MidPrice = stats.BestBid.Add(stats.BestAsk).Div(two)
		if stats.MidPrice.IsPositive() {
			hundred := decimal.NewFromInt(100)
			stats.SpreadPercent = stats.Spread.Div(stats.MidPrice).Mul(hundred)
		}
	}

	return stats
}

// createTrade 创建成交记录。
func (ob *OrderBook) createTrade(buyOrder, sellOrder *Order, qty, price decimal.Decimal, isMakerBuy bool) *Trade {
	id := ob.tradeIDGen.Add(1)
	return &Trade{
		TradeID:     fmt.Sprintf("T%d-%d", time.Now().UnixMicro(), id),
		Symbol:      ob.symbol,
		Price:       price,
		Quantity:    qty,
		BuyOrderID:  buyOrder.OrderID,
		SellOrderID: sellOrder.OrderID,
		BuyUserID:   buyOrder.UserID,
		SellUserID:  sellOrder.UserID,
		Timestamp:   time.Now().UnixNano(),
		IsMakerBuy:  isMakerBuy,
	}
}

// Clear 清空订单簿。
func (ob *OrderBook) Clear() {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.bids = NewPriceLevelMap(true)
	ob.asks = NewPriceLevelMap(false)
	ob.orders = sync.Map{}
	ob.version.Add(1)
}

// Symbol 返回订单簿标的代码。
func (ob *OrderBook) Symbol() string {
	return ob.symbol
}

// Version 返回当前版本号。
func (ob *OrderBook) Version() int64 {
	return ob.version.Load()
}
