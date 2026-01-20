package domain

import "context"

type QuoteRepository interface {
	Save(ctx context.Context, quote *Quote) error
	GetLatest(ctx context.Context, symbol string) (*Quote, error)
}

type KlineRepository interface {
	Save(ctx context.Context, kline *Kline) error
	GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*Kline, error)
}

type TradeRepository interface {
	Save(ctx context.Context, trade *Trade) error
	GetTrades(ctx context.Context, symbol string, limit int) ([]*Trade, error)
}

type OrderBookRepository interface {
	Save(ctx context.Context, ob *OrderBook) error
	Get(ctx context.Context, symbol string) (*OrderBook, error)
}
