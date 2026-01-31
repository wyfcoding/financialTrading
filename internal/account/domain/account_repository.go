package domain

import (
	"context"

	"github.com/wyfcoding/pkg/eventsourcing"
)

// AccountRepository 账户仓储接口
// 负责聚合并发控制与持久化 (Snapshot)
type AccountRepository interface {
	Save(ctx context.Context, account *Account) error
	Get(ctx context.Context, id string) (*Account, error)
	GetByUserID(ctx context.Context, userID string) ([]*Account, error)

	// ExecWithBarrier 用于TCC/Saga的屏障执行，虽然是基础设施细节，但 domain interface 往往需要感知事务边界
	ExecWithBarrier(ctx context.Context, barrier any, fn func(ctx context.Context) error) error
}

// EventStore 事件存储接口
type EventStore interface {
	Save(ctx context.Context, aggregateID string, events []eventsourcing.DomainEvent, expectedVersion int64) error
	Load(ctx context.Context, aggregateID string) ([]eventsourcing.DomainEvent, error)
}
