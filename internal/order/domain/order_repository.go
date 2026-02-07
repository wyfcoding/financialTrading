package domain

import (
	"context"

	"github.com/wyfcoding/pkg/eventsourcing"
)

// OrderRepository 订单仓储接口
type OrderRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	Save(ctx context.Context, order *Order) error
	Get(ctx context.Context, orderID string) (*Order, error)
	ListByUser(ctx context.Context, userID string, status OrderStatus, limit, offset int) ([]*Order, int64, error)
	ListBySymbol(ctx context.Context, symbol string, status OrderStatus, limit, offset int) ([]*Order, int64, error)
	GetActiveOrdersBySymbol(ctx context.Context, symbol string) ([]*Order, error)
	UpdateStatus(ctx context.Context, orderID string, status OrderStatus) error
	UpdateFilledQuantity(ctx context.Context, orderID string, filledQuantity float64) error
	Delete(ctx context.Context, orderID string) error
}

// EventStore 事件存储接口
type EventStore interface {
	Save(ctx context.Context, aggregateID string, events []eventsourcing.DomainEvent, expectedVersion int64) error
	Load(ctx context.Context, aggregateID string) ([]eventsourcing.DomainEvent, error)
}

// OrderSearchRepository 订单搜索仓储接口 (基于 ES)
type OrderSearchRepository interface {
	Index(ctx context.Context, order *Order) error
	Search(ctx context.Context, query map[string]any, limit int) ([]*Order, error)
}

// OrderReadRepository 订单读模型缓存
type OrderReadRepository interface {
	Save(ctx context.Context, order *Order) error
	Get(ctx context.Context, orderID string) (*Order, error)
}
