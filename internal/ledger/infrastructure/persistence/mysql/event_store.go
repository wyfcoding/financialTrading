// 变更说明：实现 ledger 的事件存储持久化。
package mysql

import (
	"context"
	"encoding/json"
	"time"

	"github.com/wyfcoding/financialtrading/internal/ledger/domain"
	"github.com/wyfcoding/pkg/database"
	"gorm.io/gorm"
)

type LedgerEventStore struct {
	db *database.DB
}

type EventModel struct {
	ID          uint      `gorm:"primaryKey"`
	AggregateID string    `gorm:"column:aggregate_id;type:varchar(64);index"`
	Type        string    `gorm:"column:type;type:varchar(128)"`
	Data        string    `gorm:"column:data;type:longtext"`
	Version     int64     `gorm:"column:version;index"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (EventModel) TableName() string { return "ledger_events" }

func (s *LedgerEventStore) SaveEvents(ctx context.Context, accountID string, events []domain.LedgerEvent, expectedVersion int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, event := range events {
			payload, _ := json.Marshal(event)
			model := EventModel{
				AggregateID: accountID,
				Type:        event.EventType,
				Data:        string(payload),
				Version:     event.Version,
			}
			if err := tx.Create(&model).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
