// Package application 包含订单服务的用例逻辑
package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/internal/order/domain"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"github.com/wyfcoding/financialTrading/pkg/utils"
)

// CreateOrderRequest 创建订单请求 DTO
type CreateOrderRequest struct {
	UserID        string
	Symbol        string
	Side          string
	OrderType     string
	Price         string
	Quantity      string
	TimeInForce   string
	ClientOrderID string
}

// OrderDTO 订单 DTO
type OrderDTO struct {
	OrderID        string
	UserID         string
	Symbol         string
	Side           string
	OrderType      string
	Price          string
	Quantity       string
	FilledQuantity string
	Status         string
	TimeInForce    string
	CreatedAt      int64
	UpdatedAt      int64
}

// OrderApplicationService 订单应用服务
type OrderApplicationService struct {
	orderRepo domain.OrderRepository
	snowflake *utils.SnowflakeID
}

// NewOrderApplicationService 创建订单应用服务
func NewOrderApplicationService(orderRepo domain.OrderRepository) *OrderApplicationService {
	return &OrderApplicationService{
		orderRepo: orderRepo,
		snowflake: utils.NewSnowflakeID(1),
	}
}

// CreateOrder 创建订单
// 用例流程：
// 1. 验证输入参数
// 2. 生成订单 ID
// 3. 创建订单领域对象
// 4. 保存到仓储
// 5. 发布订单创建事件（待实现）
func (oas *OrderApplicationService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderDTO, error) {
	// 记录操作开始
	defer logger.WithContext(ctx).Info("CreateOrder completed",
		"user_id", req.UserID,
		"symbol", req.Symbol,
	)

	// 验证输入
	if req.UserID == "" || req.Symbol == "" || req.Side == "" {
		return nil, fmt.Errorf("invalid request parameters")
	}

	// 解析价格和数量
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}

	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	// 生成订单 ID
	orderID := fmt.Sprintf("ORD-%d", oas.snowflake.Generate())

	// 创建订单领域对象
	order := domain.NewOrder(
		orderID,
		req.UserID,
		req.Symbol,
		domain.OrderSide(req.Side),
		domain.OrderType(req.OrderType),
		price,
		quantity,
		domain.TimeInForce(req.TimeInForce),
		req.ClientOrderID,
	)

	// 保存到仓储
	if err := oas.orderRepo.Save(ctx, order); err != nil {
		logger.WithContext(ctx).Error("Failed to save order",
			"order_id", orderID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	logger.WithContext(ctx).Debug("Order created successfully",
		"order_id", orderID,
		"user_id", req.UserID,
		"symbol", req.Symbol,
	)

	// 转换为 DTO
	return &OrderDTO{
		OrderID:        order.OrderID,
		UserID:         order.UserID,
		Symbol:         order.Symbol,
		Side:           string(order.Side),
		OrderType:      string(order.Type),
		Price:          order.Price.String(),
		Quantity:       order.Quantity.String(),
		FilledQuantity: order.FilledQuantity.String(),
		Status:         string(order.Status),
		TimeInForce:    string(order.TimeInForce),
		CreatedAt:      order.CreatedAt.Unix(),
		UpdatedAt:      order.UpdatedAt.Unix(),
	}, nil
}

// CancelOrder 取消订单
// 用例流程：
// 1. 验证订单存在且属于该用户
// 2. 检查订单是否可以取消
// 3. 更新订单状态为已取消
// 4. 发布订单取消事件（待实现）
func (oas *OrderApplicationService) CancelOrder(ctx context.Context, orderID, userID string) (*OrderDTO, error) {
	// 验证输入
	if orderID == "" || userID == "" {
		return nil, fmt.Errorf("order_id and user_id are required")
	}

	// 获取订单
	order, err := oas.orderRepo.Get(ctx, orderID)
	if err != nil {
		logger.WithContext(ctx).Error("Failed to get order",
			"order_id", orderID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	// 验证订单属于该用户
	if order.UserID != userID {
		return nil, fmt.Errorf("order does not belong to user: %s", userID)
	}

	// 检查订单是否可以取消
	if !order.CanBeCancelled() {
		return nil, fmt.Errorf("order cannot be cancelled, current status: %s", order.Status)
	}

	// 更新订单状态
	if err := oas.orderRepo.UpdateStatus(ctx, orderID, domain.OrderStatusCancelled); err != nil {
		logger.WithContext(ctx).Error("Failed to cancel order",
			"order_id", orderID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	logger.WithContext(ctx).Debug("Order cancelled successfully",
		"order_id", orderID,
	)

	// 转换为 DTO
	order.Status = domain.OrderStatusCancelled
	return &OrderDTO{
		OrderID:        order.OrderID,
		UserID:         order.UserID,
		Symbol:         order.Symbol,
		Side:           string(order.Side),
		OrderType:      string(order.Type),
		Price:          order.Price.String(),
		Quantity:       order.Quantity.String(),
		FilledQuantity: order.FilledQuantity.String(),
		Status:         string(order.Status),
		TimeInForce:    string(order.TimeInForce),
		CreatedAt:      order.CreatedAt.Unix(),
		UpdatedAt:      order.UpdatedAt.Unix(),
	}, nil
}

// GetOrder 获取订单详情
func (oas *OrderApplicationService) GetOrder(ctx context.Context, orderID, userID string) (*OrderDTO, error) {
	// 验证输入
	if orderID == "" || userID == "" {
		return nil, fmt.Errorf("order_id and user_id are required")
	}

	// 获取订单
	order, err := oas.orderRepo.Get(ctx, orderID)
	if err != nil {
		logger.WithContext(ctx).Error("Failed to get order",
			"order_id", orderID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	// 验证订单属于该用户
	if order.UserID != userID {
		return nil, fmt.Errorf("order does not belong to user: %s", userID)
	}

	// 转换为 DTO
	return &OrderDTO{
		OrderID:        order.OrderID,
		UserID:         order.UserID,
		Symbol:         order.Symbol,
		Side:           string(order.Side),
		OrderType:      string(order.Type),
		Price:          order.Price.String(),
		Quantity:       order.Quantity.String(),
		FilledQuantity: order.FilledQuantity.String(),
		Status:         string(order.Status),
		TimeInForce:    string(order.TimeInForce),
		CreatedAt:      order.CreatedAt.Unix(),
		UpdatedAt:      order.UpdatedAt.Unix(),
	}, nil
}
