package mysql

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
	"gorm.io/gorm"
)

// QuotePO
type QuotePO struct {
	ID        uint            `gorm:"primarykey"`
	Symbol    string          `gorm:"column:symbol;type:varchar(20);index;not null"`
	BidPrice  decimal.Decimal `gorm:"column:bid_price;type:decimal(32,18);not null"`
	AskPrice  decimal.Decimal `gorm:"column:ask_price;type:decimal(32,18);not null"`
	BidSize   decimal.Decimal `gorm:"column:bid_size;type:decimal(32,18);not null"`
	AskSize   decimal.Decimal `gorm:"column:ask_size;type:decimal(32,18);not null"`
	LastPrice decimal.Decimal `gorm:"column:last_price;type:decimal(32,18);not null"`
	LastSize  decimal.Decimal `gorm:"column:last_size;type:decimal(32,18);not null"`
	Timestamp time.Time       `gorm:"column:timestamp;index;not null"`
	CreatedAt time.Time
}

func (QuotePO) TableName() string { return "marketdata_quotes" }

func (po *QuotePO) ToDomain() *domain.Quote {
	return &domain.Quote{
		Symbol:    po.Symbol,
		BidPrice:  po.BidPrice,
		AskPrice:  po.AskPrice,
		BidSize:   po.BidSize,
		AskSize:   po.AskSize,
		LastPrice: po.LastPrice,
		LastSize:  po.LastSize,
		Timestamp: po.Timestamp,
	}
}

func (po *QuotePO) FromDomain(q *domain.Quote) {
	po.Symbol = q.Symbol
	po.BidPrice = q.BidPrice
	po.AskPrice = q.AskPrice
	po.BidSize = q.BidSize
	po.AskSize = q.AskSize
	po.LastPrice = q.LastPrice
	po.LastSize = q.LastSize
	po.Timestamp = q.Timestamp
}

// KlinePO
type KlinePO struct {
	ID        uint            `gorm:"primarykey"`
	Symbol    string          `gorm:"column:symbol;type:varchar(20);index:idx_symbol_interval_time;not null"`
	Interval  string          `gorm:"column:interval_str;type:varchar(10);index:idx_symbol_interval_time;not null"` // "1m", "1h"
	OpenTime  time.Time       `gorm:"column:open_time;index:idx_symbol_interval_time;not null"`
	CloseTime time.Time       `gorm:"column:close_time;not null"`
	Open      decimal.Decimal `gorm:"column:open;type:decimal(32,18);not null"`
	High      decimal.Decimal `gorm:"column:high;type:decimal(32,18);not null"`
	Low       decimal.Decimal `gorm:"column:low;type:decimal(32,18);not null"`
	Close     decimal.Decimal `gorm:"column:close;type:decimal(32,18);not null"`
	Volume    decimal.Decimal `gorm:"column:volume;type:decimal(32,18);not null"`
	CreatedAt time.Time
}

func (KlinePO) TableName() string { return "marketdata_klines" }

func (po *KlinePO) ToDomain() *domain.Kline {
	return &domain.Kline{
		Symbol:    po.Symbol,
		Interval:  po.Interval,
		OpenTime:  po.OpenTime,
		CloseTime: po.CloseTime,
		Open:      po.Open,
		High:      po.High,
		Low:       po.Low,
		Close:     po.Close,
		Volume:    po.Volume,
	}
}

func (po *KlinePO) FromDomain(k *domain.Kline) {
	po.Symbol = k.Symbol
	po.Interval = k.Interval
	po.OpenTime = k.OpenTime
	po.CloseTime = k.CloseTime
	po.Open = k.Open
	po.High = k.High
	po.Low = k.Low
	po.Close = k.Close
	po.Volume = k.Volume
}

// TradePO
type TradePO struct {
	gorm.Model
	TradeID  string          `gorm:"column:trade_id;type:varchar(32);uniqueIndex;not null"`
	Symbol   string          `gorm:"column:symbol;type:varchar(20);index;not null"`
	Price    decimal.Decimal `gorm:"column:price;type:decimal(32,18);not null"`
	Quantity decimal.Decimal `gorm:"column:quantity;type:decimal(32,18);not null"`
	Side     string          `gorm:"column:side;type:varchar(10);not null"`
	Time     time.Time       `gorm:"column:time;index;not null"`
}

func (TradePO) TableName() string { return "marketdata_trades" }

func (po *TradePO) ToDomain() *domain.Trade {
	return &domain.Trade{
		ID:        po.TradeID,
		Symbol:    po.Symbol,
		Price:     po.Price,
		Quantity:  po.Quantity,
		Side:      po.Side,
		Timestamp: po.Time,
	}
}

func (po *TradePO) FromDomain(t *domain.Trade) {
	po.TradeID = t.ID
	po.Symbol = t.Symbol
	po.Price = t.Price
	po.Quantity = t.Quantity
	po.Side = t.Side
	po.Time = t.Timestamp
}
