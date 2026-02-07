package domain

import (
	"context"
	"time"
)

// MarketDataRepository 市场数据写模型仓储接口
type MarketDataRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	// Quote
	SaveQuote(ctx context.Context, quote *Quote) error
	GetLatestQuote(ctx context.Context, symbol string) (*Quote, error)

	// Kline
	SaveKline(ctx context.Context, kline *Kline) error
	GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*Kline, error)
	GetLatestKline(ctx context.Context, symbol, interval string) (*Kline, error)

	// Trade
	SaveTrade(ctx context.Context, trade *Trade) error
	GetTrades(ctx context.Context, symbol string, limit int) ([]*Trade, error)

	// OrderBook
	SaveOrderBook(ctx context.Context, ob *OrderBook) error
	GetOrderBook(ctx context.Context, symbol string) (*OrderBook, error)
}

// QuoteReadRepository 提供报价读模型缓存
type QuoteReadRepository interface {
	Save(ctx context.Context, quote *Quote) error
	GetLatest(ctx context.Context, symbol string) (*Quote, error)
}

// KlineReadRepository 提供 K 线读模型缓存
type KlineReadRepository interface {
	Save(ctx context.Context, kline *Kline) error
	GetLatest(ctx context.Context, symbol, interval string) (*Kline, error)
	List(ctx context.Context, symbol, interval string, limit int) ([]*Kline, error)
}

// TradeReadRepository 提供成交读模型缓存
type TradeReadRepository interface {
	Save(ctx context.Context, trade *Trade) error
	List(ctx context.Context, symbol string, limit int) ([]*Trade, error)
}

// OrderBookReadRepository 提供订单簿读模型缓存
type OrderBookReadRepository interface {
	Save(ctx context.Context, ob *OrderBook) error
	Get(ctx context.Context, symbol string) (*OrderBook, error)
}

// MarketDataSearchRepository 提供基于 Elasticsearch 的行情搜索能力
type MarketDataSearchRepository interface {
	IndexQuote(ctx context.Context, quote *Quote) error
	IndexTrade(ctx context.Context, trade *Trade) error
	SearchQuotes(ctx context.Context, symbol string, startTime, endTime time.Time, limit, offset int) ([]*Quote, int64, error)
	SearchTrades(ctx context.Context, symbol string, startTime, endTime time.Time, limit, offset int) ([]*Trade, int64, error)
}

// HistoryAnalyzer 提供价格分布历史分析能力（技术实现细节在基础设施层）。
type HistoryAnalyzer interface {
	RecordTrade(price int, timestamp int64)
	QueryVolumeAtTime(timestamp int64, low, high int) int
}
