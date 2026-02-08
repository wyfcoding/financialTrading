package domain

import (
	"context"

	"github.com/wyfcoding/pkg/eventsourcing"
)

// EventStore 事件存储接口
type EventStore interface {
	Save(ctx context.Context, aggregateID string, events []eventsourcing.DomainEvent, expectedVersion int64) error
	Load(ctx context.Context, aggregateID string) ([]eventsourcing.DomainEvent, error)
}

// TradeRepository 成交单仓储接口
type TradeRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	Save(ctx context.Context, trade *Trade) error
	Get(ctx context.Context, tradeID string) (*Trade, error)
	GetByOrderID(ctx context.Context, orderID string) (*Trade, error)
	List(ctx context.Context, userID string) ([]*Trade, error)
}

// TradeSearchRepository 提供基于 Elasticsearch 的成交历史搜索
type TradeSearchRepository interface {
	Index(ctx context.Context, trade *Trade) error
	Search(ctx context.Context, userID, symbol string, limit, offset int) ([]*Trade, int64, error)
	Delete(ctx context.Context, tradeID string) error
}

// AlgoOrderRepository 算法订单仓储接口
type AlgoOrderRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	Save(ctx context.Context, order *AlgoOrder) error
	Get(ctx context.Context, algoID string) (*AlgoOrder, error)
	ListActive(ctx context.Context) ([]*AlgoOrder, error)
}

// AlgoRedisRepository 提供基于 Redis 的实时算法状态缓存
type AlgoRedisRepository interface {
	Save(ctx context.Context, order *AlgoOrder) error
	Get(ctx context.Context, algoID string) (*AlgoOrder, error)
	Delete(ctx context.Context, algoID string) error
}

// TradeReadRepository 提供基于 Redis 的成交读模型缓存
type TradeReadRepository interface {
	Save(ctx context.Context, trade *Trade) error
	GetByOrderID(ctx context.Context, orderID string) (*Trade, error)
	Delete(ctx context.Context, orderID string) error
}

// VenueRepository 交易所仓储接口
type VenueRepository interface {
	List(ctx context.Context) ([]*Venue, error)
	Get(ctx context.Context, venueID string) (*Venue, error)
}
