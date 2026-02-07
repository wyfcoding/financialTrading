package mysql

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

// QuoteModel MySQL 行情报价表映射
type QuoteModel struct {
	ID        uint            `gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time       `gorm:"column:created_at"`
	UpdatedAt time.Time       `gorm:"column:updated_at"`
	Symbol    string          `gorm:"column:symbol;type:varchar(32);index;not null;comment:标的"`
	BidPrice  decimal.Decimal `gorm:"column:bid_price;type:decimal(32,18);not null"`
	AskPrice  decimal.Decimal `gorm:"column:ask_price;type:decimal(32,18);not null"`
	BidSize   decimal.Decimal `gorm:"column:bid_size;type:decimal(32,18);not null"`
	AskSize   decimal.Decimal `gorm:"column:ask_size;type:decimal(32,18);not null"`
	LastPrice decimal.Decimal `gorm:"column:last_price;type:decimal(32,18);not null"`
	LastSize  decimal.Decimal `gorm:"column:last_size;type:decimal(32,18);not null"`
	Timestamp time.Time       `gorm:"column:timestamp;index;not null"`
}

func (QuoteModel) TableName() string { return "quotes" }

// KlineModel MySQL K 线表映射
type KlineModel struct {
	ID        uint            `gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time       `gorm:"column:created_at"`
	UpdatedAt time.Time       `gorm:"column:updated_at"`
	Symbol    string          `gorm:"column:symbol;type:varchar(32);index;not null"`
	Interval  string          `gorm:"column:interval_period;type:varchar(10);index;not null"`
	OpenTime  time.Time       `gorm:"column:open_time;index;not null"`
	CloseTime time.Time       `gorm:"column:close_time;not null"`
	Open      decimal.Decimal `gorm:"column:open;type:decimal(32,18);not null"`
	High      decimal.Decimal `gorm:"column:high;type:decimal(32,18);not null"`
	Low       decimal.Decimal `gorm:"column:low;type:decimal(32,18);not null"`
	Close     decimal.Decimal `gorm:"column:close;type:decimal(32,18);not null"`
	Volume    decimal.Decimal `gorm:"column:volume;type:decimal(32,18);not null"`
}

func (KlineModel) TableName() string { return "klines" }

// TradeModel MySQL 成交记录表映射
type TradeModel struct {
	ID        string          `gorm:"primaryKey;type:varchar(64);column:id"`
	CreatedAt time.Time       `gorm:"column:created_at"`
	UpdatedAt time.Time       `gorm:"column:updated_at"`
	Symbol    string          `gorm:"column:symbol;type:varchar(32);index;not null"`
	Price     decimal.Decimal `gorm:"column:price;type:decimal(32,18);not null"`
	Quantity  decimal.Decimal `gorm:"column:quantity;type:decimal(32,18);not null"`
	Side      string          `gorm:"column:side;type:varchar(10);not null"`
	Timestamp time.Time       `gorm:"column:timestamp;index;not null"`
}

func (TradeModel) TableName() string { return "trades" }

// OrderBookModel MySQL 订单簿表映射
type OrderBookModel struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
	Symbol    string    `gorm:"column:symbol;type:varchar(32);uniqueIndex;not null"`
	BidsJSON  string    `gorm:"column:bids;type:json;not null"`
	AsksJSON  string    `gorm:"column:asks;type:json;not null"`
	Timestamp time.Time `gorm:"column:timestamp;not null"`
}

func (OrderBookModel) TableName() string { return "order_books" }

// --- mapping helpers ---

type orderBookLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

func toQuoteModel(q *domain.Quote) *QuoteModel {
	if q == nil {
		return nil
	}
	return &QuoteModel{
		Symbol:    q.Symbol,
		BidPrice:  q.BidPrice,
		AskPrice:  q.AskPrice,
		BidSize:   q.BidSize,
		AskSize:   q.AskSize,
		LastPrice: q.LastPrice,
		LastSize:  q.LastSize,
		Timestamp: q.Timestamp,
	}
}

func toQuote(m *QuoteModel) *domain.Quote {
	if m == nil {
		return nil
	}
	return &domain.Quote{
		Symbol:    m.Symbol,
		BidPrice:  m.BidPrice,
		AskPrice:  m.AskPrice,
		BidSize:   m.BidSize,
		AskSize:   m.AskSize,
		LastPrice: m.LastPrice,
		LastSize:  m.LastSize,
		Timestamp: m.Timestamp,
	}
}

func toKlineModel(k *domain.Kline) *KlineModel {
	if k == nil {
		return nil
	}
	return &KlineModel{
		Symbol:    k.Symbol,
		Interval:  k.Interval,
		OpenTime:  k.OpenTime,
		CloseTime: k.CloseTime,
		Open:      k.Open,
		High:      k.High,
		Low:       k.Low,
		Close:     k.Close,
		Volume:    k.Volume,
	}
}

func toKline(m *KlineModel) *domain.Kline {
	if m == nil {
		return nil
	}
	return &domain.Kline{
		Symbol:    m.Symbol,
		Interval:  m.Interval,
		OpenTime:  m.OpenTime,
		CloseTime: m.CloseTime,
		Open:      m.Open,
		High:      m.High,
		Low:       m.Low,
		Close:     m.Close,
		Volume:    m.Volume,
	}
}

func toTradeModel(t *domain.Trade) *TradeModel {
	if t == nil {
		return nil
	}
	return &TradeModel{
		ID:        t.ID,
		Symbol:    t.Symbol,
		Price:     t.Price,
		Quantity:  t.Quantity,
		Side:      t.Side,
		Timestamp: t.Timestamp,
	}
}

func toTrade(m *TradeModel) *domain.Trade {
	if m == nil {
		return nil
	}
	return &domain.Trade{
		ID:        m.ID,
		Symbol:    m.Symbol,
		Price:     m.Price,
		Quantity:  m.Quantity,
		Side:      m.Side,
		Timestamp: m.Timestamp,
	}
}

func toOrderBookModel(ob *domain.OrderBook) (*OrderBookModel, error) {
	if ob == nil {
		return nil, nil
	}
	bids := make([]orderBookLevel, 0, len(ob.Bids))
	for _, bid := range ob.Bids {
		bids = append(bids, orderBookLevel{Price: bid.Price.String(), Quantity: bid.Quantity.String()})
	}
	asks := make([]orderBookLevel, 0, len(ob.Asks))
	for _, ask := range ob.Asks {
		asks = append(asks, orderBookLevel{Price: ask.Price.String(), Quantity: ask.Quantity.String()})
	}
	bidsJSON, err := json.Marshal(bids)
	if err != nil {
		return nil, err
	}
	asksJSON, err := json.Marshal(asks)
	if err != nil {
		return nil, err
	}
	return &OrderBookModel{
		Symbol:    ob.Symbol,
		BidsJSON:  string(bidsJSON),
		AsksJSON:  string(asksJSON),
		Timestamp: ob.Timestamp,
	}, nil
}

func toOrderBook(m *OrderBookModel) (*domain.OrderBook, error) {
	if m == nil {
		return nil, nil
	}
	var bids []orderBookLevel
	var asks []orderBookLevel
	if err := json.Unmarshal([]byte(m.BidsJSON), &bids); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(m.AsksJSON), &asks); err != nil {
		return nil, err
	}
	bidItems := make([]domain.OrderBookItem, 0, len(bids))
	for _, bid := range bids {
		px, _ := decimal.NewFromString(bid.Price)
		qty, _ := decimal.NewFromString(bid.Quantity)
		bidItems = append(bidItems, domain.OrderBookItem{Price: px, Quantity: qty})
	}
	askItems := make([]domain.OrderBookItem, 0, len(asks))
	for _, ask := range asks {
		px, _ := decimal.NewFromString(ask.Price)
		qty, _ := decimal.NewFromString(ask.Quantity)
		askItems = append(askItems, domain.OrderBookItem{Price: px, Quantity: qty})
	}
	return &domain.OrderBook{
		Symbol:    m.Symbol,
		Bids:      bidItems,
		Asks:      askItems,
		Timestamp: m.Timestamp,
	}, nil
}
