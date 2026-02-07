package domain

import "time"

const (
	QuoteUpdatedEventType     = "marketdata.quote.updated"
	KlineUpdatedEventType     = "marketdata.kline.updated"
	TradeExecutedEventType    = "marketdata.trade.executed"
	OrderBookUpdatedEventType = "marketdata.orderbook.updated"
)

// QuoteUpdatedEvent 报价更新事件
type QuoteUpdatedEvent struct {
	Symbol    string    `json:"symbol"`
	BidPrice  string    `json:"bid_price"`
	AskPrice  string    `json:"ask_price"`
	BidSize   string    `json:"bid_size"`
	AskSize   string    `json:"ask_size"`
	LastPrice string    `json:"last_price"`
	LastSize  string    `json:"last_size"`
	Timestamp time.Time `json:"timestamp"`
}

// KlineUpdatedEvent K线更新事件
type KlineUpdatedEvent struct {
	Symbol     string    `json:"symbol"`
	Interval   string    `json:"interval"`
	OpenPrice  string    `json:"open_price"`
	HighPrice  string    `json:"high_price"`
	LowPrice   string    `json:"low_price"`
	ClosePrice string    `json:"close_price"`
	Volume     string    `json:"volume"`
	OpenTime   time.Time `json:"open_time"`
	CloseTime  time.Time `json:"close_time"`
	Timestamp  time.Time `json:"timestamp"`
}

// TradeExecutedEvent 交易执行事件
type TradeExecutedEvent struct {
	Symbol    string    `json:"symbol"`
	Price     string    `json:"price"`
	Quantity  string    `json:"quantity"`
	Side      string    `json:"side"`
	TradeID   string    `json:"trade_id"`
	Timestamp time.Time `json:"timestamp"`
}

// OrderBookUpdatedEvent 订单簿更新事件
type OrderBookUpdatedEvent struct {
	Symbol    string      `json:"symbol"`
	Bids      [][2]string `json:"bids"`
	Asks      [][2]string `json:"asks"`
	Timestamp time.Time   `json:"timestamp"`
}
