package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/eventsourcing"
	"gorm.io/gorm"
)

type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository 创建并返回一个新的 orderRepository 实例。
func NewOrderRepository(db *gorm.DB) domain.OrderRepository {
	return &orderRepository{db: db}
}

// --- tx helpers ---

func (r *orderRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *orderRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *orderRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *orderRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *orderRepository) Save(ctx context.Context, order *domain.Order) error {
	model := toOrderModel(order)
	if model == nil {
		return nil
	}
	db := r.getDB(ctx).WithContext(ctx)
	if model.ID == 0 {
		if err := db.Create(model).Error; err != nil {
			return err
		}
		order.ID = model.ID
		order.CreatedAt = model.CreatedAt
		order.UpdatedAt = model.UpdatedAt
		return nil
	}
	return db.Model(&OrderModel{}).
		Where("order_id = ?", model.OrderID).
		Updates(map[string]any{
			"user_id":         model.UserID,
			"symbol":          model.Symbol,
			"side":            model.Side,
			"type":            model.Type,
			"price":           model.Price,
			"stop_price":      model.StopPrice,
			"quantity":        model.Quantity,
			"filled_quantity": model.FilledQuantity,
			"average_price":   model.AveragePrice,
			"status":          model.Status,
			"tif":             model.TimeInForce,
			"parent_id":       model.ParentOrderID,
			"is_oco":          model.IsOCO,
			"updated_at":      time.Now(),
		}).Error
}

func (r *orderRepository) Get(ctx context.Context, orderID string) (*domain.Order, error) {
	var model OrderModel
	if err := r.getDB(ctx).WithContext(ctx).Where("order_id = ?", orderID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	order := toOrder(&model)
	if order != nil {
		order.SetID(order.OrderID)
	}
	return order, nil
}

func (r *orderRepository) ListByUser(ctx context.Context, userID string, status domain.OrderStatus, limit, offset int) ([]*domain.Order, int64, error) {
	var models []OrderModel
	var total int64
	query := r.getDB(ctx).WithContext(ctx).Model(&OrderModel{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", string(status))
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Limit(limit).Offset(offset).Order("created_at desc").Find(&models).Error; err != nil {
		return nil, 0, err
	}
	orders := make([]*domain.Order, len(models))
	for i := range models {
		orders[i] = toOrder(&models[i])
		if orders[i] != nil {
			orders[i].SetID(orders[i].OrderID)
		}
	}
	return orders, total, nil
}

func (r *orderRepository) ListBySymbol(ctx context.Context, symbol string, status domain.OrderStatus, limit, offset int) ([]*domain.Order, int64, error) {
	var models []OrderModel
	var total int64
	query := r.getDB(ctx).WithContext(ctx).Model(&OrderModel{}).Where("symbol = ?", symbol)
	if status != "" {
		query = query.Where("status = ?", string(status))
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Limit(limit).Offset(offset).Order("created_at desc").Find(&models).Error; err != nil {
		return nil, 0, err
	}
	orders := make([]*domain.Order, len(models))
	for i := range models {
		orders[i] = toOrder(&models[i])
		if orders[i] != nil {
			orders[i].SetID(orders[i].OrderID)
		}
	}
	return orders, total, nil
}

func (r *orderRepository) GetActiveOrdersBySymbol(ctx context.Context, symbol string) ([]*domain.Order, error) {
	var models []OrderModel
	statuses := []string{string(domain.StatusPending), string(domain.StatusValidated), string(domain.StatusPartiallyFilled)}
	if err := r.getDB(ctx).WithContext(ctx).
		Where("symbol = ? AND status IN ?", symbol, statuses).
		Find(&models).Error; err != nil {
		return nil, err
	}
	orders := make([]*domain.Order, len(models))
	for i := range models {
		orders[i] = toOrder(&models[i])
		if orders[i] != nil {
			orders[i].SetID(orders[i].OrderID)
		}
	}
	return orders, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, orderID string, status domain.OrderStatus) error {
	return r.getDB(ctx).WithContext(ctx).
		Model(&OrderModel{}).
		Where("order_id = ?", orderID).
		Updates(map[string]any{"status": string(status), "updated_at": time.Now()}).Error
}

func (r *orderRepository) UpdateFilledQuantity(ctx context.Context, orderID string, filledQuantity float64) error {
	return r.getDB(ctx).WithContext(ctx).
		Model(&OrderModel{}).
		Where("order_id = ?", orderID).
		Updates(map[string]any{"filled_quantity": filledQuantity, "updated_at": time.Now()}).Error
}

func (r *orderRepository) Delete(ctx context.Context, orderID string) error {
	return r.getDB(ctx).WithContext(ctx).Where("order_id = ?", orderID).Delete(&OrderModel{}).Error
}

func (r *orderRepository) getDB(ctx context.Context) *gorm.DB {
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
		po := &EventModel{
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
