package application

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/order/domain"
)

// PlaceOrderCommand 下单命令
type PlaceOrderCommand struct {
	UserID       string
	Symbol       string
	Side         string
	Type         string
	Price        float64
	StopPrice    float64
	Quantity     float64
	TimeInForce  string
	ParentOrderID string
	IsOCO        bool
}

// CancelOrderCommand 取消订单命令
type CancelOrderCommand struct {
	OrderID string
	UserID  string
	Reason  string
}

// OrderCommand 处理订单相关的命令操作
type OrderCommand struct {
	repo          domain.OrderRepository
	eventPublisher domain.EventPublisher
}

// NewOrderCommand 创建新的 OrderCommand 实例
func NewOrderCommand(repo domain.OrderRepository, eventPublisher domain.EventPublisher) *OrderCommand {
	return &OrderCommand{
		repo:          repo,
		eventPublisher: eventPublisher,
	}
}

// PlaceOrder 下单
func (c *OrderCommand) PlaceOrder(ctx context.Context, cmd PlaceOrderCommand) (string, error) {
	// 创建订单
	order := domain.NewOrder(
		generateOrderID(),
		cmd.UserID,
		cmd.Symbol,
		domain.OrderSide(cmd.Side),
		domain.OrderType(cmd.Type),
		cmd.Price,
		cmd.Quantity,
	)

	order.StopPrice = cmd.StopPrice
	order.TimeInForce = domain.TimeInForce(cmd.TimeInForce)
	order.ParentOrderID = cmd.ParentOrderID
	order.IsOCO = cmd.IsOCO

	// 验证订单
	if err := order.Validate(); err != nil {
		// 发布订单被拒绝事件
		rejectedEvent := domain.OrderRejectedEvent{
			OrderID:     order.OrderID,
			UserID:      order.UserID,
			Symbol:      order.Symbol,
			Reason:      err.Error(),
			RejectedAt:  time.Now().Unix(),
			OccurredOn:  time.Now(),
		}

		c.eventPublisher.PublishOrderRejected(rejectedEvent)
		return "", err
	}

	// 保存订单
	if err := c.repo.Save(ctx, order); err != nil {
		return "", err
	}

	// 发布订单创建事件
	createdEvent := domain.OrderCreatedEvent{
		OrderID:       order.OrderID,
		UserID:        order.UserID,
		Symbol:        order.Symbol,
		Side:          order.Side,
		Type:          order.Type,
		Price:         order.Price,
		StopPrice:     order.StopPrice,
		Quantity:      order.Quantity,
		TimeInForce:   order.TimeInForce,
		ParentOrderID: order.ParentOrderID,
		IsOCO:         order.IsOCO,
		OccurredOn:    time.Now(),
	}

	c.eventPublisher.PublishOrderCreated(createdEvent)

	// 验证订单并更新状态
	order.MarkValidated()
	if err := c.repo.Save(ctx, order); err != nil {
		return order.OrderID, err
	}

	// 发布订单验证通过事件
	validatedEvent := domain.OrderValidatedEvent{
		OrderID:     order.OrderID,
		UserID:      order.UserID,
		Symbol:      order.Symbol,
		ValidatedAt: time.Now().Unix(),
		OccurredOn:  time.Now(),
	}

	c.eventPublisher.PublishOrderValidated(validatedEvent)

	// 发布订单状态变更事件
	statusChangedEvent := domain.OrderStatusChangedEvent{
		OrderID:     order.OrderID,
		OldStatus:   domain.StatusPending,
		NewStatus:   domain.StatusValidated,
		UpdatedAt:   time.Now().Unix(),
		OccurredOn:  time.Now(),
	}

	c.eventPublisher.PublishOrderStatusChanged(statusChangedEvent)

	return order.OrderID, nil
}

// CancelOrder 取消订单
func (c *OrderCommand) CancelOrder(ctx context.Context, cmd CancelOrderCommand) error {
	// 获取订单
	order, err := c.repo.Get(ctx, cmd.OrderID)
	if err != nil {
		return err
	}

	// 检查用户权限
	if order.UserID != cmd.UserID {
		return ErrUnauthorized
	}

	// 检查订单状态
	if order.Status != domain.StatusValidated && order.Status != domain.StatusPartiallyFilled {
		return ErrInvalidOrderStatus
	}

	oldStatus := order.Status

	// 更新订单状态
	order.Status = domain.StatusCancelled
	order.UpdatedAt = time.Now()

	if err := c.repo.Save(ctx, order); err != nil {
		return err
	}

	// 发布订单被取消事件
	cancelledEvent := domain.OrderCancelledEvent{
		OrderID:     order.OrderID,
		UserID:      order.UserID,
		Symbol:      order.Symbol,
		Reason:      cmd.Reason,
		CancelledAt: time.Now().Unix(),
		OccurredOn:  time.Now(),
	}

	c.eventPublisher.PublishOrderCancelled(cancelledEvent)

	// 发布订单状态变更事件
	statusChangedEvent := domain.OrderStatusChangedEvent{
		OrderID:     order.OrderID,
		OldStatus:   oldStatus,
		NewStatus:   domain.StatusCancelled,
		UpdatedAt:   time.Now().Unix(),
		OccurredOn:  time.Now(),
	}

	c.eventPublisher.PublishOrderStatusChanged(statusChangedEvent)

	return nil
}

// UpdateOrderExecution 更新订单执行状态
func (c *OrderCommand) UpdateOrderExecution(ctx context.Context, orderID string, filledQty, tradePrice float64) error {
	// 获取订单
	order, err := c.repo.Get(ctx, orderID)
	if err != nil {
		return err
	}

	oldStatus := order.Status

	// 更新订单执行状态
	order.UpdateExecution(filledQty, tradePrice)

	if err := c.repo.Save(ctx, order); err != nil {
		return err
	}

	// 根据新状态发布相应事件
	switch order.Status {
	case domain.StatusPartiallyFilled:
		// 发布订单部分成交事件
		partialEvent := domain.OrderPartiallyFilledEvent{
			OrderID:         order.OrderID,
			UserID:          order.UserID,
			Symbol:          order.Symbol,
			FilledQuantity:  order.FilledQuantity,
			RemainingQuantity: order.Quantity - order.FilledQuantity,
			TradePrice:      tradePrice,
			AveragePrice:    order.AveragePrice,
			FilledAt:        time.Now().Unix(),
			OccurredOn:      time.Now(),
		}

		c.eventPublisher.PublishOrderPartiallyFilled(partialEvent)

	case domain.StatusFilled:
		// 发布订单完全成交事件
		filledEvent := domain.OrderFilledEvent{
			OrderID:       order.OrderID,
			UserID:        order.UserID,
			Symbol:        order.Symbol,
			TotalQuantity: order.Quantity,
			AveragePrice:  order.AveragePrice,
			FilledAt:      time.Now().Unix(),
			OccurredOn:    time.Now(),
		}

		c.eventPublisher.PublishOrderFilled(filledEvent)
	}

	// 发布订单状态变更事件
	statusChangedEvent := domain.OrderStatusChangedEvent{
		OrderID:     order.OrderID,
		OldStatus:   oldStatus,
		NewStatus:   order.Status,
		UpdatedAt:   time.Now().Unix(),
		OccurredOn:  time.Now(),
	}

	c.eventPublisher.PublishOrderStatusChanged(statusChangedEvent)

	return nil
}

// 生成订单 ID
func generateOrderID() string {
	return "ORDER_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// 生成随机字符串
func randomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[time.Now().UnixNano()%int64(len(letterBytes))]
	}
	return string(b)
}

// 错误定义
var (
	ErrUnauthorized     = NewError("unauthorized", "unauthorized to cancel this order")
	ErrInvalidOrderStatus = NewError("invalid_order_status", "order status cannot be cancelled")
)

// Error 自定义错误
type Error struct {
	Code    string
	Message string
}

// Error 实现 error 接口
func (e *Error) Error() string {
	return e.Message
}

// NewError 创建新的错误
func NewError(code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}
