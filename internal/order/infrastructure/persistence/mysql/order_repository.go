// Package mysql 提供了订单仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OrderModel 订单数据库模型
type OrderModel struct {
	gorm.Model
	OrderID        string `gorm:"column:order_id;type:varchar(32);uniqueIndex;not null"`
	UserID         string `gorm:"column:user_id;type:varchar(32);index;not null"`
	Symbol         string `gorm:"column:symbol;type:varchar(20);index;not null"`
	Side           string `gorm:"column:side;type:varchar(10);not null"`
	Type           string `gorm:"column:type;type:varchar(20);not null"`
	Price          string `gorm:"column:price;type:decimal(32,18);not null"`
	Quantity       string `gorm:"column:quantity;type:decimal(32,18);not null"`
	FilledQuantity string `gorm:"column:filled_quantity;type:decimal(32,18);default:'0';not null"`
	Status         string `gorm:"column:status;type:varchar(20);index;not null"`
	TimeInForce    string `gorm:"column:time_in_force;type:varchar(10);not null"`
	ClientOrderID  string `gorm:"column:client_order_id;type:varchar(32);index"`
	Remark         string `gorm:"column:remark;type:varchar(255)"`
}

// TableName 指定表名
func (OrderModel) TableName() string {
	return "orders"
}

// orderRepositoryImpl 是 domain.OrderRepository 接口的 GORM 实现。
type orderRepositoryImpl struct {
	db *gorm.DB
}

// NewOrderRepository 创建订单仓储实例
func NewOrderRepository(db *gorm.DB) domain.OrderRepository {
	return &orderRepositoryImpl{
		db: db,
	}
}

// Save 实现 domain.OrderRepository.Save
func (r *orderRepositoryImpl) Save(ctx context.Context, order *domain.Order) error {
	model := &OrderModel{
		Model:          order.Model,
		OrderID:        order.OrderID,
		UserID:         order.UserID,
		Symbol:         order.Symbol,
		Side:           string(order.Side),
		Type:           string(order.Type),
		Price:          order.Price.String(),
		Quantity:       order.Quantity.String(),
		FilledQuantity: order.FilledQuantity.String(),
		Status:         string(order.Status),
		TimeInForce:    string(order.TimeInForce),
		ClientOrderID:  order.ClientOrderID,
		Remark:         order.Remark,
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "order_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "filled_quantity", "remark"}),
	}).Create(model).Error
	if err != nil {
		logging.Error(ctx, "order_repository.Save failed", "order_id", order.OrderID, "error", err)
		return fmt.Errorf("failed to save order: %w", err)
	}

	order.Model = model.Model
	return nil
}

// Get 实现 domain.OrderRepository.Get
func (r *orderRepositoryImpl) Get(ctx context.Context, orderID string) (*domain.Order, error) {
	var model OrderModel
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "order_repository.Get failed", "order_id", orderID, "error", err)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return r.toDomain(&model), nil
}

// ListByUser 实现 domain.OrderRepository.ListByUser
func (r *orderRepositoryImpl) ListByUser(ctx context.Context, userID string, status domain.OrderStatus, limit, offset int) ([]*domain.Order, int64, error) {
	var models []OrderModel
	var total int64
	db := r.db.WithContext(ctx).Model(&OrderModel{}).Where("user_id = ?", userID)
	if status != "" {
		db = db.Where("status = ?", string(status))
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		logging.Error(ctx, "order_repository.ListByUser failed", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}

	orders := make([]*domain.Order, len(models))
	for i, m := range models {
		orders[i] = r.toDomain(&m)
	}
	return orders, total, nil
}

// ListBySymbol 实现 domain.OrderRepository.ListBySymbol
func (r *orderRepositoryImpl) ListBySymbol(ctx context.Context, symbol string, status domain.OrderStatus, limit, offset int) ([]*domain.Order, int64, error) {
	var models []OrderModel
	var total int64
	db := r.db.WithContext(ctx).Model(&OrderModel{}).Where("symbol = ?", symbol)
	if status != "" {
		db = db.Where("status = ?", string(status))
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		logging.Error(ctx, "order_repository.ListBySymbol failed", "symbol", symbol, "error", err)
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}

	orders := make([]*domain.Order, len(models))
	for i, m := range models {
		orders[i] = r.toDomain(&m)
	}
	return orders, total, nil
}

// UpdateStatus 实现 domain.OrderRepository.UpdateStatus
func (r *orderRepositoryImpl) UpdateStatus(ctx context.Context, orderID string, status domain.OrderStatus) error {
	err := r.db.WithContext(ctx).Model(&OrderModel{}).Where("order_id = ?", orderID).Update("status", string(status)).Error
	if err != nil {
		logging.Error(ctx, "order_repository.UpdateStatus failed", "order_id", orderID, "error", err)
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}

// UpdateFilledQuantity 实现 domain.OrderRepository.UpdateFilledQuantity
func (r *orderRepositoryImpl) UpdateFilledQuantity(ctx context.Context, orderID string, filledQuantity decimal.Decimal) error {
	err := r.db.WithContext(ctx).Model(&OrderModel{}).Where("order_id = ?", orderID).Update("filled_quantity", filledQuantity.String()).Error
	if err != nil {
		logging.Error(ctx, "order_repository.UpdateFilledQuantity failed", "order_id", orderID, "error", err)
		return fmt.Errorf("failed to update filled quantity: %w", err)
	}
	return nil
}

// Delete 实现 domain.OrderRepository.Delete
func (r *orderRepositoryImpl) Delete(ctx context.Context, orderID string) error {
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Delete(&OrderModel{}).Error; err != nil {
		logging.Error(ctx, "order_repository.Delete failed", "order_id", orderID, "error", err)
		return fmt.Errorf("failed to delete order: %w", err)
	}
	return nil
}

func (r *orderRepositoryImpl) toDomain(m *OrderModel) *domain.Order {
	price, _ := decimal.NewFromString(m.Price)
	qty, _ := decimal.NewFromString(m.Quantity)
	filled, _ := decimal.NewFromString(m.FilledQuantity)

	return &domain.Order{
		Model:          m.Model,
		OrderID:        m.OrderID,
		UserID:         m.UserID,
		Symbol:         m.Symbol,
		Side:           domain.OrderSide(m.Side),
		Type:           domain.OrderType(m.Type),
		Price:          price,
		Quantity:       qty,
		FilledQuantity: filled,
		Status:         domain.OrderStatus(m.Status),
		TimeInForce:    domain.TimeInForce(m.TimeInForce),
		ClientOrderID:  m.ClientOrderID,
		Remark:         m.Remark,
	}
}
