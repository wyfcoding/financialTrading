package mysql

import (
	"context"
	"encoding/json"

	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type EventStore struct {
	db *gorm.DB
}

func NewEventStore(db *gorm.DB) *EventStore {
	return &EventStore{db: db}
}

func (s *EventStore) Append(ctx context.Context, aggregateID string, events []domain.AccountEvent) error {
	db := s.getDB(ctx)

	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}

		po := &EventPO{
			AggregateID: aggregateID,
			EventType:   event.EventType(),
			Payload:     string(payload),
			OccurredAt:  event.OccurredAt().UnixNano(),
		}

		if err := db.Create(po).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *EventStore) Load(ctx context.Context, aggregateID string) ([]domain.AccountEvent, error) {
	// 在混合架构中，通常我们直接从 Snapshot (AccountPO) 读取状态，
	// EventStore 主要用于重放审计或 CQRS Projection 构建。
	// 这里暂未实现完整的 Event 反序列化工厂，因为当前 Application Layer 也可以直接读 Repo。
	return nil, nil
}

func (s *EventStore) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return s.db
}
