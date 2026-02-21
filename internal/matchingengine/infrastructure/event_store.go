// Package infrastructure 事件存储实现
package infrastructure

import (
	"context"
	"encoding/json"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"gorm.io/gorm"
)

type EventStoreModel struct {
	ID          uint64 `gorm:"primaryKey"`
	AggregateID string `gorm:"index"`
	EventType   string
	Payload     []byte
	Version     int64
	CreatedAt   int64 `gorm:"autoCreateTime:milli"`
}

type GormEventStore struct {
	db *gorm.DB
}

func (s *GormEventStore) SaveEvents(ctx context.Context, aggregateID string, events []domain.MatchingEvent, expectedVersion int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i, event := range events {
			payload, _ := json.Marshal(event)
			model := &EventStoreModel{
				AggregateID: aggregateID,
				EventType:   event.EventType(),
				Payload:     payload,
				Version:     expectedVersion + int64(i) + 1,
			}
			if err := tx.Create(model).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *GormEventStore) LoadEvents(ctx context.Context, aggregateID string) ([]domain.MatchingEvent, error) {
	var models []EventStoreModel
	if err := s.db.Where("aggregate_id = ?", aggregateID).Order("version asc").Find(&models).Error; err != nil {
		return nil, err
	}

	var events []domain.MatchingEvent
	for _, m := range models {
		// 根据 EventType 反序列化为具体的领域事件 (需工厂模式适配)
		// ... 逻辑省略
		_ = m
	}
	return events, nil
}
