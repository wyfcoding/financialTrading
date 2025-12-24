// 包 订单服务的用例逻辑
package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
)

// CreateOrderRequest 创建订单请求 DTO
// 用于接收创建订单的请求参数
type CreateOrderRequest struct {
	UserID        string // 用户 ID
	Symbol        string // 交易对符号
	Side          string // 买卖方向
	OrderType     string // 订单类型
	Price         string // 价格（限价单必填）
	Quantity      string // 数量
	TimeInForce   string // 有效期策略
	ClientOrderID string // 客户端订单 ID（幂等性）
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
}

// NewOrderApplicationService 创建订单应用服务
func NewOrderApplicationService(orderRepo domain.OrderRepository) *OrderApplicationService {
	return &OrderApplicationService{
		orderRepo: orderRepo,
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
	// 记录性能监控
	defer logging.LogDuration(ctx, "Order creation completed",
		"user_id", req.UserID,
		"symbol", req.Symbol,
	)()

	logging.Info(ctx, "Creating new order",
		"user_id", req.UserID,
		"symbol", req.Symbol,
		"side", req.Side,
		"order_type", req.OrderType,
	)

	// 验证输入
	if req.UserID == "" || req.Symbol == "" || req.Side == "" {
		logging.Warn(ctx, "Invalid order creation parameters",
			"user_id", req.UserID,
			"symbol", req.Symbol,
			"side", req.Side,
		)
		return nil, fmt.Errorf("invalid request parameters")
	}

	// 解析价格和数量
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		logging.Error(ctx, "Failed to parse order price",
			"user_id", req.UserID,
			"price", req.Price,
			"error", err,
		)
		return nil, fmt.Errorf("invalid price: %w", err)
	}

	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		logging.Error(ctx, "Failed to parse order quantity",
			"user_id", req.UserID,
			"quantity", req.Quantity,
			"error", err,
		)
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	// 生成订单 ID
	orderID := fmt.Sprintf("ORD-%d", idgen.GenID())

	logging.Debug(ctx, "Generated order ID",
		"order_id", orderID,
		"user_id", req.UserID,
	)

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
		logging.Error(ctx, "Failed to save order to repository",
			"order_id", orderID,
			"user_id", req.UserID,
			"symbol", req.Symbol,
			"error", err,
		)
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	logging.Info(ctx, "Order created successfully",
		"order_id", orderID,
		"user_id", req.UserID,
		"symbol", req.Symbol,
		"side", req.Side,
		"price", price.String(),
		"quantity", quantity.String(),
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
		logging.Error(ctx, "Failed to get order",
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
		logging.Error(ctx, "Failed to update order status to cancelled",
			"order_id", orderID,
			"user_id", userID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	logging.Info(ctx, "Order cancelled successfully",
		"order_id", orderID,
		"user_id", userID,
		"symbol", order.Symbol,
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
		logging.Error(ctx, "Failed to get order",
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
