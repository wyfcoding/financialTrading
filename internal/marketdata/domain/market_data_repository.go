package domain

import (
	"context"
)

// QuoteRepository 行情数据仓储接口
type QuoteRepository interface {
	Save(ctx context.Context, quote *Quote) error
	GetLatest(ctx context.Context, symbol string) (*Quote, error)
	GetHistory(ctx context.Context, symbol string, startTime, endTime int64) ([]*Quote, error)
	DeleteExpired(ctx context.Context, beforeTime int64) error
}

// KlineRepository K 线数据仓储接口
type KlineRepository interface {
	Save(ctx context.Context, kline *Kline) error
	Get(ctx context.Context, symbol, interval string, startTime, endTime int64) ([]*Kline, error)
	GetLatest(ctx context.Context, symbol, interval string, limit int) ([]*Kline, error)
	DeleteExpired(ctx context.Context, beforeTime int64) error
}

// TradeRepository 交易记录仓储接口
type TradeRepository interface {
	Save(ctx context.Context, trade *Trade) error
	GetHistory(ctx context.Context, symbol string, startTime, endTime int64, limit int) ([]*Trade, error)
	GetLatest(ctx context.Context, symbol string, limit int) ([]*Trade, error)
	DeleteExpired(ctx context.Context, beforeTime int64) error
}

// OrderBookRepository 订单簿仓储接口 (内存或缓存实现)
type OrderBookRepository interface {
	Save(ctx context.Context, orderBook *OrderBook) error
	GetLatest(ctx context.Context, symbol string) (*OrderBook, error)
}
