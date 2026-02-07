package domain

import "context"

// TradeRepository 成交记录仓储接口
type TradeRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	Save(ctx context.Context, trade *Trade) error
	GetLatestTrades(ctx context.Context, symbol string, limit int) ([]*Trade, error)
}

// OrderBookRepository 订单簿仓储接口
type OrderBookRepository interface {
	SaveSnapshot(ctx context.Context, snapshot *OrderBookSnapshot) error
}

// TradeReadRepository 交易读模型缓存
type TradeReadRepository interface {
	Save(ctx context.Context, trade *Trade) error
	List(ctx context.Context, symbol string, limit int) ([]*Trade, error)
}

// OrderBookReadRepository 订单簿读模型缓存
type OrderBookReadRepository interface {
	Save(ctx context.Context, snapshot *OrderBookSnapshot, depth int) error
	Get(ctx context.Context, symbol string, depth int) (*OrderBookSnapshot, error)
}

// TradeSearchRepository 提供基于 Elasticsearch 的成交搜索
type TradeSearchRepository interface {
	Index(ctx context.Context, trade *Trade) error
	Search(ctx context.Context, symbol string, limit, offset int) ([]*Trade, int64, error)
	Delete(ctx context.Context, tradeID string) error
}
