package repository

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/internal/order/domain"
	"github.com/wyfcoding/financialTrading/pkg/db"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"gorm.io/gorm"
)

// OrderModel 订单数据库模型
// 对应数据库中的 orders 表
type OrderModel struct {
	gorm.Model
	// 订单 ID，业务主键，唯一索引
	OrderID string `gorm:"column:order_id;type:varchar(50);uniqueIndex;not null" json:"order_id"`
	// 用户 ID，普通索引
	UserID string `gorm:"column:user_id;type:varchar(50);index;not null" json:"user_id"`
	// 交易对符号，普通索引
	Symbol string `gorm:"column:symbol;type:varchar(50);index;not null" json:"symbol"`
	// 买卖方向
	Side string `gorm:"column:side;type:varchar(10);not null" json:"side"`
	// 订单类型
	Type string `gorm:"column:type;type:varchar(20);not null" json:"type"`
	// 价格
	Price string `gorm:"column:price;type:decimal(20,8);not null" json:"price"`
	// 数量
	Quantity string `gorm:"column:quantity;type:decimal(20,8);not null" json:"quantity"`
	// 已成交数量
	FilledQuantity string `gorm:"column:filled_quantity;type:decimal(20,8);not null" json:"filled_quantity"`
	// 订单状态
	Status string `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
	// 有效期
	TimeInForce string `gorm:"column:time_in_force;type:varchar(10);not null" json:"time_in_force"`
	// 客户端订单 ID
	ClientOrderID string `gorm:"column:client_order_id;type:varchar(100);index" json:"client_order_id"`
	// 备注
	Remark string `gorm:"column:remark;type:text" json:"remark"`
}

// TableName 指定表名
func (OrderModel) TableName() string {
	return "orders"
}

// OrderRepositoryImpl 订单仓储实现
type OrderRepositoryImpl struct {
	db *db.DB
}

// NewOrderRepository 创建订单仓储
func NewOrderRepository(database *db.DB) domain.OrderRepository {
	return &OrderRepositoryImpl{
		db: database,
	}
}

// Save 保存订单
func (or *OrderRepositoryImpl) Save(ctx context.Context, order *domain.Order) error {
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

	if err := or.db.WithContext(ctx).Create(model).Error; err != nil {
		logger.Error(ctx, "Failed to save order",
			"order_id", order.OrderID,
			"error", err,
		)
		return fmt.Errorf("failed to save order: %w", err)
	}

	// 更新 domain 对象的 Model 信息 (ID, CreatedAt, UpdatedAt)
	order.Model = model.Model

	return nil
}

// Get 获取订单
func (or *OrderRepositoryImpl) Get(ctx context.Context, orderID string) (*domain.Order, error) {
	var model OrderModel

	if err := or.db.WithContext(ctx).Where("order_id = ?", orderID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Error(ctx, "Failed to get order",
			"order_id", orderID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return or.modelToDomain(&model), nil
}

// ListByUser 获取用户订单列表
func (or *OrderRepositoryImpl) ListByUser(ctx context.Context, userID string, status domain.OrderStatus, limit, offset int) ([]*domain.Order, int64, error) {
	var models []OrderModel
	var total int64

	query := or.db.WithContext(ctx).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", string(status))
	}

	if err := query.Model(&OrderModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&models).Error; err != nil {
		logger.Error(ctx, "Failed to list orders",
			"user_id", userID,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}

	orders := make([]*domain.Order, 0, len(models))
	for _, model := range models {
		orders = append(orders, or.modelToDomain(&model))
	}

	return orders, total, nil
}

// ListBySymbol 获取交易对订单列表
func (or *OrderRepositoryImpl) ListBySymbol(ctx context.Context, symbol string, status domain.OrderStatus, limit, offset int) ([]*domain.Order, int64, error) {
	var models []OrderModel
	var total int64

	query := or.db.WithContext(ctx).Where("symbol = ?", symbol)
	if status != "" {
		query = query.Where("status = ?", string(status))
	}

	if err := query.Model(&OrderModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&models).Error; err != nil {
		logger.Error(ctx, "Failed to list orders",
			"symbol", symbol,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}

	orders := make([]*domain.Order, 0, len(models))
	for _, model := range models {
		orders = append(orders, or.modelToDomain(&model))
	}

	return orders, total, nil
}

// UpdateStatus 更新订单状态
func (or *OrderRepositoryImpl) UpdateStatus(ctx context.Context, orderID string, status domain.OrderStatus) error {
	if err := or.db.WithContext(ctx).Model(&OrderModel{}).Where("order_id = ?", orderID).Update("status", string(status)).Error; err != nil {
		logger.Error(ctx, "Failed to update order status",
			"order_id", orderID,
			"error", err,
		)
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

// UpdateFilledQuantity 更新已成交数量
func (or *OrderRepositoryImpl) UpdateFilledQuantity(ctx context.Context, orderID string, filledQuantity decimal.Decimal) error {
	if err := or.db.WithContext(ctx).Model(&OrderModel{}).Where("order_id = ?", orderID).Update("filled_quantity", filledQuantity.String()).Error; err != nil {
		logger.Error(ctx, "Failed to update filled quantity",
			"order_id", orderID,
			"error", err,
		)
		return fmt.Errorf("failed to update filled quantity: %w", err)
	}

	return nil
}

// Delete 删除订单
func (or *OrderRepositoryImpl) Delete(ctx context.Context, orderID string) error {
	if err := or.db.WithContext(ctx).Where("order_id = ?", orderID).Delete(&OrderModel{}).Error; err != nil {
		logger.Error(ctx, "Failed to delete order",
			"order_id", orderID,
			"error", err,
		)
		return fmt.Errorf("failed to delete order: %w", err)
	}

	return nil
}

// modelToDomain 将数据库模型转换为领域对象
func (or *OrderRepositoryImpl) modelToDomain(model *OrderModel) *domain.Order {
	price, _ := decimal.NewFromString(model.Price)
	quantity, _ := decimal.NewFromString(model.Quantity)
	filledQuantity, _ := decimal.NewFromString(model.FilledQuantity)

	return &domain.Order{
		Model:          model.Model,
		OrderID:        model.OrderID,
		UserID:         model.UserID,
		Symbol:         model.Symbol,
		Side:           domain.OrderSide(model.Side),
		Type:           domain.OrderType(model.Type),
		Price:          price,
		Quantity:       quantity,
		FilledQuantity: filledQuantity,
		Status:         domain.OrderStatus(model.Status),
		TimeInForce:    domain.TimeInForce(model.TimeInForce),
		ClientOrderID:  model.ClientOrderID,
		Remark:         model.Remark,
	}
}
