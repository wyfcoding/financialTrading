package domain

import "context"

// MarketDataRepository 市场数据统一仓储接口
type MarketDataRepository interface {
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
