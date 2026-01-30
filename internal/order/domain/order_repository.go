package domain

import (
	"context"
)

// OrderRepository 订单仓储接口
type OrderRepository interface {
	// Save 保存或更新订单
	Save(ctx context.Context, order *Order) error
	// Get 根据订单 ID 获取订单
	Get(ctx context.Context, orderID string) (*Order, error)
	// ListByUser 获取用户订单列表
	ListByUser(ctx context.Context, userID string, status OrderStatus, limit, offset int) ([]*Order, int64, error)
	// ListBySymbol 获取交易对订单列表
	ListBySymbol(ctx context.Context, symbol string, status OrderStatus, limit, offset int) ([]*Order, int64, error)
	// GetActiveOrdersBySymbol 获取指定交易对的所有活跃订单 (OPEN, PARTIALLY_FILLED)
	GetActiveOrdersBySymbol(ctx context.Context, symbol string) ([]*Order, error)
	// UpdateStatus 更新订单状态
	UpdateStatus(ctx context.Context, orderID string, status OrderStatus) error
	// UpdateFilledQuantity 更新已成交数量
	UpdateFilledQuantity(ctx context.Context, orderID string, filledQuantity float64) error
	// Delete 根据订单 ID 删除订单
	Delete(ctx context.Context, orderID string) error
}
