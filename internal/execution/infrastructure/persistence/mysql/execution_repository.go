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

// --- tx helpers ---

func (r *tradeRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *tradeRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *tradeRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *tradeRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *tradeRepository) Save(ctx context.Context, t *domain.Trade) error {
	db := r.getDB(ctx)
	model := toTradeModel(t)
	if model.ID == 0 {
		if err := db.WithContext(ctx).Create(model).Error; err != nil {
			return err
		}
		t.ID = model.ID
		t.CreatedAt = model.CreatedAt
		t.UpdatedAt = model.UpdatedAt
		return nil
	}
	return db.WithContext(ctx).
		Model(&TradeModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"trade_id":    model.TradeID,
			"order_id":    model.OrderID,
			"user_id":     model.UserID,
			"symbol":      model.Symbol,
			"side":        model.Side,
			"price":       model.ExecutedPrice,
			"quantity":    model.ExecutedQuantity,
			"executed_at": model.ExecutedAt,
			"status":      model.Status,
		}).Error
}

func (r *tradeRepository) Get(ctx context.Context, tradeID string) (*domain.Trade, error) {
	var model TradeModel
	if err := r.getDB(ctx).WithContext(ctx).Where("trade_id = ?", tradeID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toTrade(&model), nil
}

func (r *tradeRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Trade, error) {
	var model TradeModel
	if err := r.getDB(ctx).WithContext(ctx).Where("order_id = ?", orderID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toTrade(&model), nil
}

func (r *tradeRepository) List(ctx context.Context, userID string) ([]*domain.Trade, error) {
	var models []*TradeModel
	if err := r.getDB(ctx).WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, err
	}
	trades := make([]*domain.Trade, len(models))
	for i, m := range models {
		trades[i] = toTrade(m)
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

// --- tx helpers ---

func (r *algoOrderRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *algoOrderRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *algoOrderRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *algoOrderRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *algoOrderRepository) Save(ctx context.Context, o *domain.AlgoOrder) error {
	db := r.getDB(ctx)
	model := toAlgoOrderModel(o)
	if model.ID == 0 {
		if err := db.WithContext(ctx).Create(model).Error; err != nil {
			return err
		}
		o.ID = model.ID
		o.CreatedAt = model.CreatedAt
		o.UpdatedAt = model.UpdatedAt
		return nil
	}
	return db.WithContext(ctx).
		Model(&AlgoOrderModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"algo_id":            model.AlgoID,
			"user_id":            model.UserID,
			"symbol":             model.Symbol,
			"side":               model.Side,
			"total_quantity":     model.TotalQuantity,
			"executed_qty":       model.ExecutedQuantity,
			"participation_rate": model.ParticipationRate,
			"algo_type":          model.AlgoType,
			"start_time":         model.StartTime,
			"end_time":           model.EndTime,
			"status":             model.Status,
			"strategy_params":    model.StrategyParams,
		}).Error
}

func (r *algoOrderRepository) Get(ctx context.Context, algoID string) (*domain.AlgoOrder, error) {
	var model AlgoOrderModel
	if err := r.getDB(ctx).WithContext(ctx).Where("algo_id = ?", algoID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toAlgoOrder(&model), nil
}

func (r *algoOrderRepository) ListActive(ctx context.Context) ([]*domain.AlgoOrder, error) {
	var models []*AlgoOrderModel
	if err := r.getDB(ctx).WithContext(ctx).Where("status = ?", "RUNNING").Find(&models).Error; err != nil {
		return nil, err
	}
	orders := make([]*domain.AlgoOrder, len(models))
	for i, m := range models {
		orders[i] = toAlgoOrder(m)
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

func (s *eventStore) Load(ctx context.Context, aggregateID string) ([]eventsourcing.DomainEvent, error) {
	return nil, nil // TODO: 实现加载逻辑
}

func (s *eventStore) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return s.db
}
