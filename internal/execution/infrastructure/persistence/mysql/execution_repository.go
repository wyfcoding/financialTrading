package mysql

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/eventsourcing"
	"gorm.io/gorm"
)

type tradeRepository struct {
	db *gorm.DB
}

// NewTradeRepository 创建并返回一个新的 tradeRepository 实例。
func NewTradeRepository(db *gorm.DB) domain.TradeRepository {
	return &tradeRepository{db: db}
}

func (r *tradeRepository) Save(ctx context.Context, t *domain.Trade) error {
	db := r.getDB(ctx)
	if t.Model.ID == 0 {
		return db.Create(t).Error
	}
	return db.Save(t).Error
}

func (r *tradeRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Trade, error) {
	var trade domain.Trade
	if err := r.getDB(ctx).Where("order_id = ?", orderID).First(&trade).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	trade.SetID(trade.TradeID)
	return &trade, nil
}

func (r *tradeRepository) List(ctx context.Context, userID string) ([]*domain.Trade, error) {
	var trades []*domain.Trade
	if err := r.getDB(ctx).Where("user_id = ?", userID).Find(&trades).Error; err != nil {
		return nil, err
	}
	for _, t := range trades {
		t.SetID(t.TradeID)
	}
	return trades, nil
}

func (r *tradeRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}

type algoOrderRepository struct {
	db *gorm.DB
}

// NewAlgoOrderRepository 创建并返回一个新的 algoOrderRepository 实例。
func NewAlgoOrderRepository(db *gorm.DB) domain.AlgoOrderRepository {
	return &algoOrderRepository{db: db}
}

func (r *algoOrderRepository) Save(ctx context.Context, o *domain.AlgoOrder) error {
	db := r.getDB(ctx)
	if o.Model.ID == 0 {
		return db.Create(o).Error
	}
	return db.Save(o).Error
}

func (r *algoOrderRepository) Get(ctx context.Context, algoID string) (*domain.AlgoOrder, error) {
	var order domain.AlgoOrder
	if err := r.getDB(ctx).Where("algo_id = ?", algoID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	order.SetID(order.AlgoID)
	return &order, nil
}

func (r *algoOrderRepository) ListActive(ctx context.Context) ([]*domain.AlgoOrder, error) {
	var orders []*domain.AlgoOrder
	if err := r.getDB(ctx).Where("status = ?", "RUNNING").Find(&orders).Error; err != nil {
		return nil, err
	}
	for _, o := range orders {
		o.SetID(o.AlgoID)
	}
	return orders, nil
}

func (r *algoOrderRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}

// --- EventStore 实现 ---

type eventStore struct {
	db *gorm.DB
}

func NewEventStore(db *gorm.DB) domain.EventStore {
	return &eventStore{db: db}
}

func (s *eventStore) Save(ctx context.Context, aggregateID string, events []eventsourcing.DomainEvent, expectedVersion int64) error {
	db := s.getDB(ctx)
	for _, event := range events {
		payload, _ := json.Marshal(event)
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

func (s *eventStore) Load(ctx context.Context, aggregateID string) ([]eventsourcing.DomainEvent, error) {
	return nil, nil // TODO: 实现加载逻辑
}

func (s *eventStore) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return s.db
}

type EventPO struct {
	gorm.Model
	AggregateID string `gorm:"column:aggregate_id;type:varchar(64);index;not null"`
	EventType   string `gorm:"column:event_type;type:varchar(50);not null"`
	Payload     string `gorm:"column:payload;type:json;not null"`
	OccurredAt  int64  `gorm:"column:occurred_at;not null"`
}

func (EventPO) TableName() string {
	return "execution_events"
}
