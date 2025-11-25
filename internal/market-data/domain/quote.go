// Package domain 包含市场数据服务的领域模型、实体、聚合、值对象、领域服务、仓储接口
package domain

import (
	"github.com/shopspring/decimal"
)

// Quote 行情数据实体
// 代表某个交易对在某个时刻的最新行情信息
type Quote struct {
	// 交易对符号（如 BTC/USDT）
	Symbol string
	// 买价
	BidPrice decimal.Decimal
	// 卖价
	AskPrice decimal.Decimal
	// 买量
	BidSize decimal.Decimal
	// 卖量
	AskSize decimal.Decimal
	// 最后成交价
	LastPrice decimal.Decimal
	// 最后成交量
	LastSize decimal.Decimal
	// 时间戳（毫秒）
	Timestamp int64
	// 数据来源（如 exchange_name）
	Source string
}

// NewQuote 创建行情数据
func NewQuote(symbol string, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize decimal.Decimal, timestamp int64, source string) *Quote {
	return &Quote{
		Symbol:    symbol,
		BidPrice:  bidPrice,
		AskPrice:  askPrice,
		BidSize:   bidSize,
		AskSize:   askSize,
		LastPrice: lastPrice,
		LastSize:  lastSize,
		Timestamp: timestamp,
		Source:    source,
	}
}

// GetSpread 获取买卖价差
func (q *Quote) GetSpread() decimal.Decimal {
	return q.AskPrice.Sub(q.BidPrice)
}

// GetMidPrice 获取中间价
func (q *Quote) GetMidPrice() decimal.Decimal {
	return q.BidPrice.Add(q.AskPrice).Div(decimal.NewFromInt(2))
}

// Kline K 线数据实体
// 代表某个交易对在某个时间周期内的 OHLCV 数据
type Kline struct {
	// 交易对符号
	Symbol string
	// 时间周期（1m, 5m, 15m, 1h, 4h, 1d）
	Interval string
	// 开盘时间（毫秒）
	OpenTime int64
	// 开盘价
	Open decimal.Decimal
	// 最高价
	High decimal.Decimal
	// 最低价
	Low decimal.Decimal
	// 收盘价
	Close decimal.Decimal
	// 成交量
	Volume decimal.Decimal
	// 收盘时间（毫秒）
	CloseTime int64
	// 成交额
	QuoteAssetVolume decimal.Decimal
	// 成交笔数
	TradeCount int64
	// 主动买入成交量
	TakerBuyBaseAssetVolume decimal.Decimal
	// 主动买入成交额
	TakerBuyQuoteAssetVolume decimal.Decimal
}

// NewKline 创建 K 线数据
func NewKline(symbol, interval string, openTime int64, open, high, low, close, volume decimal.Decimal, closeTime int64) *Kline {
	return &Kline{
		Symbol:    symbol,
		Interval:  interval,
		OpenTime:  openTime,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
		CloseTime: closeTime,
	}
}

// GetChange 获取涨跌幅（百分比）
func (k *Kline) GetChange() decimal.Decimal {
	if k.Open.IsZero() {
		return decimal.Zero
	}
	return k.Close.Sub(k.Open).Div(k.Open).Mul(decimal.NewFromInt(100))
}

// OrderBook 订单簿实体
// 代表某个交易对的当前订单簿快照
type OrderBook struct {
	// 交易对符号
	Symbol string
	// 买单列表（按价格从高到低）
	Bids []*OrderBookLevel
	// 卖单列表（按价格从低到高）
	Asks []*OrderBookLevel
	// 时间戳（毫秒）
	Timestamp int64
	// 数据来源
	Source string
}

// OrderBookLevel 订单簿层级
type OrderBookLevel struct {
	// 价格
	Price decimal.Decimal
	// 数量
	Quantity decimal.Decimal
}

// NewOrderBook 创建订单簿
func NewOrderBook(symbol string, timestamp int64, source string) *OrderBook {
	return &OrderBook{
		Symbol:    symbol,
		Bids:      make([]*OrderBookLevel, 0),
		Asks:      make([]*OrderBookLevel, 0),
		Timestamp: timestamp,
		Source:    source,
	}
}

// AddBid 添加买单层级
func (ob *OrderBook) AddBid(price, quantity decimal.Decimal) {
	ob.Bids = append(ob.Bids, &OrderBookLevel{
		Price:    price,
		Quantity: quantity,
	})
}

// AddAsk 添加卖单层级
func (ob *OrderBook) AddAsk(price, quantity decimal.Decimal) {
	ob.Asks = append(ob.Asks, &OrderBookLevel{
		Price:    price,
		Quantity: quantity,
	})
}

// GetBestBid 获取最优买价
func (ob *OrderBook) GetBestBid() *OrderBookLevel {
	if len(ob.Bids) == 0 {
		return nil
	}
	return ob.Bids[0]
}

// GetBestAsk 获取最优卖价
func (ob *OrderBook) GetBestAsk() *OrderBookLevel {
	if len(ob.Asks) == 0 {
		return nil
	}
	return ob.Asks[0]
}

// Trade 交易记录实体
// 代表某个交易对的一笔成交记录
type Trade struct {
	// 交易 ID
	TradeID string
	// 交易对符号
	Symbol string
	// 成交价格
	Price decimal.Decimal
	// 成交数量
	Quantity decimal.Decimal
	// 买卖方向（BUY 或 SELL）
	Side string
	// 时间戳（毫秒）
	Timestamp int64
	// 数据来源
	Source string
}

// NewTrade 创建交易记录
func NewTrade(tradeID, symbol string, price, quantity decimal.Decimal, side string, timestamp int64, source string) *Trade {
	return &Trade{
		TradeID:   tradeID,
		Symbol:    symbol,
		Price:     price,
		Quantity:  quantity,
		Side:      side,
		Timestamp: timestamp,
		Source:    source,
	}
}

// GetTradeValue 获取交易额
func (t *Trade) GetTradeValue() decimal.Decimal {
	return t.Price.Mul(t.Quantity)
}

// QuoteRepository 行情数据仓储接口
type QuoteRepository interface {
	// 保存行情数据
	Save(quote *Quote) error
	// 获取最新行情
	GetLatest(symbol string) (*Quote, error)
	// 获取历史行情
	GetHistory(symbol string, startTime, endTime int64) ([]*Quote, error)
	// 删除过期行情
	DeleteExpired(beforeTime int64) error
}

// KlineRepository K 线数据仓储接口
type KlineRepository interface {
	// 保存 K 线数据
	Save(kline *Kline) error
	// 获取 K 线数据
	Get(symbol, interval string, startTime, endTime int64) ([]*Kline, error)
	// 获取最新 K 线
	GetLatest(symbol, interval string, limit int) ([]*Kline, error)
	// 删除过期 K 线
	DeleteExpired(beforeTime int64) error
}

// OrderBookRepository 订单簿仓储接口
type OrderBookRepository interface {
	// 保存订单簿快照
	Save(orderBook *OrderBook) error
	// 获取最新订单簿
	GetLatest(symbol string) (*OrderBook, error)
	// 获取历史订单簿
	GetHistory(symbol string, startTime, endTime int64) ([]*OrderBook, error)
	// 删除过期订单簿
	DeleteExpired(beforeTime int64) error
}

// TradeRepository 交易记录仓储接口
type TradeRepository interface {
	// 保存交易记录
	Save(trade *Trade) error
	// 获取交易历史
	GetHistory(symbol string, startTime, endTime int64, limit int) ([]*Trade, error)
	// 获取最新交易
	GetLatest(symbol string, limit int) ([]*Trade, error)
	// 删除过期交易记录
	DeleteExpired(beforeTime int64) error
}

// MarketDataService 市场数据领域服务
// 提供市场数据相关的业务逻辑
type MarketDataService interface {
	// 计算技术指标（如 MA、RSI 等）
	CalculateTechnicalIndicators(klines []*Kline) map[string]interface{}
	// 检测异常行情
	DetectAnomalies(quote *Quote, historicalQuotes []*Quote) bool
	// 计算波动率
	CalculateVolatility(klines []*Kline) decimal.Decimal
}
