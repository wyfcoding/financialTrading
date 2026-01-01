// Package domain 市场数据服务的领域模型、实体、聚合、值对象、领域服务、仓储接口
package domain

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Quote 行情数据实体
type Quote struct {
	gorm.Model
	// Symbol 交易对符号（如 BTC/USDT）
	Symbol string `gorm:"column:symbol;type:varchar(20);index;not null"`
	// BidPrice 买价
	BidPrice decimal.Decimal `gorm:"column:bid_price;type:decimal(32,18);not null"`
	// AskPrice 卖价
	AskPrice decimal.Decimal `gorm:"column:ask_price;type:decimal(32,18);not null"`
	// BidSize 买量
	BidSize decimal.Decimal `gorm:"column:bid_size;type:decimal(32,18);not null"`
	// AskSize 卖量
	AskSize decimal.Decimal `gorm:"column:ask_size;type:decimal(32,18);not null"`
	// LastPrice 最后成交价
	LastPrice decimal.Decimal `gorm:"column:last_price;type:decimal(32,18);not null"`
	// LastSize 最后成交量
	LastSize decimal.Decimal `gorm:"column:last_size;type:decimal(32,18);not null"`
	// Timestamp 时间戳（毫秒）
	Timestamp int64 `gorm:"column:timestamp;type:bigint;not null"`
	// Source 数据来源
	Source string `gorm:"column:source;type:varchar(50)"`
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
type Kline struct {
	gorm.Model
	// Symbol 交易对符号
	Symbol string `gorm:"column:symbol;type:varchar(20);index;not null"`
	// Interval 时间周期
	Interval string `gorm:"column:interval;type:varchar(10);not null"`
	// OpenTime 开盘时间
	OpenTime int64 `gorm:"column:open_time;type:bigint;not null"`
	// Open 开盘价
	Open decimal.Decimal `gorm:"column:open_price;type:decimal(32,18);not null"`
	// High 最高价
	High decimal.Decimal `gorm:"column:high_price;type:decimal(32,18);not null"`
	// Low 最低价
	Low decimal.Decimal `gorm:"column:low_price;type:decimal(32,18);not null"`
	// Close 收盘价
	Close decimal.Decimal `gorm:"column:close_price;type:decimal(32,18);not null"`
	// Volume 成交量
	Volume decimal.Decimal `gorm:"column:volume;type:decimal(32,18);not null"`
	// CloseTime 收盘时间
	CloseTime int64 `gorm:"column:close_time;type:bigint;not null"`
	// QuoteAssetVolume 成交额
	QuoteAssetVolume decimal.Decimal `gorm:"column:quote_asset_volume;type:decimal(32,18);not null"`
	// TradeCount 成交笔数
	TradeCount int64 `gorm:"column:trade_count;type:bigint;not null"`
	// TakerBuyBaseAssetVolume 主动买入成交量
	TakerBuyBaseAssetVolume decimal.Decimal `gorm:"column:taker_buy_base_volume;type:decimal(32,18);not null"`
	// TakerBuyQuoteAssetVolume 主动买入成交额
	TakerBuyQuoteAssetVolume decimal.Decimal `gorm:"column:taker_buy_quote_volume;type:decimal(32,18);not null"`
}

// GetChange 获取涨跌幅（百分比）
func (k *Kline) GetChange() decimal.Decimal {
	if k.Open.IsZero() {
		return decimal.Zero
	}
	return k.Close.Sub(k.Open).Div(k.Open).Mul(decimal.NewFromInt(100))
}

// Trade 交易记录实体
type Trade struct {
	gorm.Model
	// TradeID 交易 ID (外部)
	TradeID string `gorm:"column:trade_id;type:varchar(32);uniqueIndex;not null"`
	// Symbol 交易对符号
	Symbol string `gorm:"column:symbol;type:varchar(20);index;not null"`
	// Price 成交价格
	Price decimal.Decimal `gorm:"column:price;type:decimal(32,18);not null"`
	// Quantity 成交数量
	Quantity decimal.Decimal `gorm:"column:quantity;type:decimal(32,18);not null"`
	// Side 买卖方向
	Side string `gorm:"column:side;type:varchar(10);not null"`
	// Timestamp 时间戳
	Timestamp int64 `gorm:"column:timestamp;type:bigint;not null"`
	// Source 数据来源
	Source string `gorm:"column:source;type:varchar(50)"`
}

// GetTradeValue 获取交易额
func (t *Trade) GetTradeValue() decimal.Decimal {
	return t.Price.Mul(t.Quantity)
}

// OrderBookLevel 订单簿层级 (值对象)
type OrderBookLevel struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

// OrderBook 订单簿实体
type OrderBook struct {
	Symbol    string
	Bids      []*OrderBookLevel
	Asks      []*OrderBookLevel
	Timestamp int64
	Source    string
}

// End of domain file
