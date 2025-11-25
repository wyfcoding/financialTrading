// Package domain 包含订单服务的领域模型
package domain

import (
	"context"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusPending         OrderStatus = "PENDING"
	OrderStatusOpen            OrderStatus = "OPEN"
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	OrderStatusFilled          OrderStatus = "FILLED"
	OrderStatusCancelled       OrderStatus = "CANCELLED"
	OrderStatusRejected        OrderStatus = "REJECTED"
)

// OrderSide 订单方向
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderType 订单类型
type OrderType string

const (
	OrderTypeLimit      OrderType = "LIMIT"
	OrderTypeMarket     OrderType = "MARKET"
	OrderTypeStopLoss   OrderType = "STOP_LOSS"
	OrderTypeTakeProfit OrderType = "TAKE_PROFIT"
)

// TimeInForce 订单有效期
type TimeInForce string

const (
	TimeInForceGTC TimeInForce = "GTC" // Good Till Cancel
	TimeInForceIOC TimeInForce = "IOC" // Immediate Or Cancel
	TimeInForceFOK TimeInForce = "FOK" // Fill Or Kill
)

// Order 订单实体
// 代表用户提交的一笔订单
type Order struct {
	gorm.Model
	// 订单 ID
	OrderID string `gorm:"column:order_id;type:varchar(32);uniqueIndex;not null" json:"order_id"`
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(32);index;not null" json:"user_id"`
	// 交易对符号
	Symbol string `gorm:"column:symbol;type:varchar(20);index;not null" json:"symbol"`
	// 买卖方向
	Side OrderSide `gorm:"column:side;type:varchar(10);not null" json:"side"`
	// 订单类型
	Type OrderType `gorm:"column:type;type:varchar(20);not null" json:"type"`
	// 价格
	Price decimal.Decimal `gorm:"column:price;type:decimal(20,8);not null" json:"price"`
	// 数量
	Quantity decimal.Decimal `gorm:"column:quantity;type:decimal(20,8);not null" json:"quantity"`
	// 已成交数量
	FilledQuantity decimal.Decimal `gorm:"column:filled_quantity;type:decimal(20,8);not null;default:0" json:"filled_quantity"`
	// 订单状态
	Status OrderStatus `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
	// 有效期
	TimeInForce TimeInForce `gorm:"column:time_in_force;type:varchar(10);not null" json:"time_in_force"`
	// 客户端订单 ID（用于幂等性）
	ClientOrderID string `gorm:"column:client_order_id;type:varchar(32);index" json:"client_order_id"`
	// 备注
	Remark string `gorm:"column:remark;type:varchar(255)" json:"remark"`
}

// NewOrder 创建订单
func NewOrder(orderID, userID, symbol string, side OrderSide, orderType OrderType, price, quantity decimal.Decimal, timeInForce TimeInForce, clientOrderID string) *Order {
	return &Order{
		OrderID:        orderID,
		UserID:         userID,
		Symbol:         symbol,
		Side:           side,
		Type:           orderType,
		Price:          price,
		Quantity:       quantity,
		FilledQuantity: decimal.Zero,
		Status:         OrderStatusPending,
		TimeInForce:    timeInForce,
		ClientOrderID:  clientOrderID,
	}
}

// GetRemainingQuantity 获取剩余数量
func (o *Order) GetRemainingQuantity() decimal.Decimal {
	return o.Quantity.Sub(o.FilledQuantity)
}

// IsFilled 是否已完全成交
func (o *Order) IsFilled() bool {
	return o.FilledQuantity.Equal(o.Quantity)
}

// CanBeCancelled 是否可以取消
func (o *Order) CanBeCancelled() bool {
	return o.Status == OrderStatusOpen || o.Status == OrderStatusPartiallyFilled
}

// OrderRepository 订单仓储接口
type OrderRepository interface {
	// 保存订单
	Save(ctx context.Context, order *Order) error
	// 获取订单
	Get(ctx context.Context, orderID string) (*Order, error)
	// 获取用户订单列表
	ListByUser(ctx context.Context, userID string, status OrderStatus, limit, offset int) ([]*Order, int64, error)
	// 获取交易对订单列表
	ListBySymbol(ctx context.Context, symbol string, status OrderStatus, limit, offset int) ([]*Order, int64, error)
	// 更新订单状态
	UpdateStatus(ctx context.Context, orderID string, status OrderStatus) error
	// 更新已成交数量
	UpdateFilledQuantity(ctx context.Context, orderID string, filledQuantity decimal.Decimal) error
	// 删除订单
	Delete(ctx context.Context, orderID string) error
}

// OrderDomainService 订单领域服务
type OrderDomainService interface {
	// 验证订单
	ValidateOrder(order *Order) error
	// 计算订单费用
	CalculateFee(order *Order) decimal.Decimal
	// 检查用户余额
	CheckBalance(userID string, amount decimal.Decimal) (bool, error)
}
