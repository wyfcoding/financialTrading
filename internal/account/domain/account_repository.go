package domain

import (
	"context"

	"github.com/wyfcoding/pkg/eventsourcing"
)

// AccountRepository 账户仓储接口
// 负责聚合并发控制与持久化 (Snapshot)
type AccountRepository interface {
	// --- tx helpers ---
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	Save(ctx context.Context, account *Account) error
	Get(ctx context.Context, id string) (*Account, error)
	GetByUserID(ctx context.Context, userID string) ([]*Account, error)

	// ExecWithBarrier 用于TCC/Saga的屏障执行
	ExecWithBarrier(ctx context.Context, barrier any, fn func(ctx context.Context) error) error
}

// EventStore 事件存储接口
type EventStore interface {
	Save(ctx context.Context, aggregateID string, events []eventsourcing.DomainEvent, expectedVersion int64) error
	Load(ctx context.Context, aggregateID string) ([]eventsourcing.DomainEvent, error)
}

// AccountReadRepository 账户读模型仓储（Redis）。
type AccountReadRepository interface {
	Save(ctx context.Context, account *Account) error
	Get(ctx context.Context, accountID string) (*Account, error)
	Delete(ctx context.Context, accountID string) error
}
