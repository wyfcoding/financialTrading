package persistence

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository 创建并返回一个新的 orderRepository 实例。
func NewOrderRepository(db *gorm.DB) domain.OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Save(ctx context.Context, order *domain.Order) error {
	db := r.getDB(ctx)
	if order.ID == 0 {
		return db.Create(order).Error
	}
	return db.Save(order).Error
}

func (r *orderRepository) Get(ctx context.Context, orderID string) (*domain.Order, error) {
	var order domain.Order
	if err := r.getDB(ctx).Where("order_id = ?", orderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) ListByUser(ctx context.Context, userID string, status domain.OrderStatus, limit, offset int) ([]*domain.Order, int64, error) {
	var orders []*domain.Order
	var total int64
	db := r.getDB(ctx).Model(&domain.Order{}).Where("user_id = ?", userID)
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Limit(limit).Offset(offset).Order("created_at desc").Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func (r *orderRepository) ListBySymbol(ctx context.Context, symbol string, status domain.OrderStatus, limit, offset int) ([]*domain.Order, int64, error) {
	var orders []*domain.Order
	var total int64
	db := r.getDB(ctx).Model(&domain.Order{}).Where("symbol = ?", symbol)
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Limit(limit).Offset(offset).Order("created_at desc").Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func (r *orderRepository) GetActiveOrdersBySymbol(ctx context.Context, symbol string) ([]*domain.Order, error) {
	var orders []*domain.Order
	err := r.getDB(ctx).Where("symbol = ? AND status IN ?", symbol, []domain.OrderStatus{domain.StatusPending, domain.StatusValidated, domain.StatusPartiallyFilled}).Find(&orders).Error
	return orders, err
}

func (r *orderRepository) UpdateStatus(ctx context.Context, orderID string, status domain.OrderStatus) error {
	return r.getDB(ctx).Model(&domain.Order{}).Where("order_id = ?", orderID).Update("status", status).Error
}

func (r *orderRepository) UpdateFilledQuantity(ctx context.Context, orderID string, filledQuantity float64) error {
	return r.getDB(ctx).Model(&domain.Order{}).Where("order_id = ?", orderID).Update("filled_quantity", filledQuantity).Error
}

func (r *orderRepository) Delete(ctx context.Context, orderID string) error {
	return r.getDB(ctx).Where("order_id = ?", orderID).Delete(&domain.Order{}).Error
}

func (r *orderRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
