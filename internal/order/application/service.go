package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/financialtrading/internal/order/infrastructure/messaging"
	"gorm.io/gorm"
)

// mockEventPublisher 事件发布者的空实现
type mockEventPublisher struct{}

func (m *mockEventPublisher) PublishOrderCreated(event domain.OrderCreatedEvent) error { return nil }
func (m *mockEventPublisher) PublishOrderValidated(event domain.OrderValidatedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishOrderRejected(event domain.OrderRejectedEvent) error { return nil }
func (m *mockEventPublisher) PublishOrderPartiallyFilled(event domain.OrderPartiallyFilledEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishOrderFilled(event domain.OrderFilledEvent) error { return nil }
func (m *mockEventPublisher) PublishOrderCancelled(event domain.OrderCancelledEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishOrderExpired(event domain.OrderExpiredEvent) error { return nil }
func (m *mockEventPublisher) PublishOrderStatusChanged(event domain.OrderStatusChangedEvent) error {
	return nil
}

// OrderService 订单服务门面，整合命令和查询服务
type OrderService struct {
	Command   *OrderCommand
	Query     *OrderQueryService
	DTMServer string
}

// NewOrderService 构造函数
func NewOrderService(repo domain.OrderRepository, db interface{}) (*OrderService, error) {
	// 创建事件发布者
	var eventPublisher domain.EventPublisher
	if gormDB, ok := db.(*gorm.DB); ok {
		eventPublisher = messaging.NewOutboxEventPublisher(gormDB)
	} else {
		// 使用空实现作为降级方案
		eventPublisher = &mockEventPublisher{}
	}

	// 创建命令服务
	command := NewOrderCommand(repo, eventPublisher)

	// 创建查询服务
	query := NewOrderQueryService(repo)

	return &OrderService{
		Command: command,
		Query:   query,
	}, nil
}

// --- Command (Writes) ---

// PlaceOrder 下单
func (s *OrderService) PlaceOrder(ctx context.Context, cmd PlaceOrderCommand) (string, error) {
	return s.Command.PlaceOrder(ctx, cmd)
}

// CancelOrder 取消订单
func (s *OrderService) CancelOrder(ctx context.Context, cmd CancelOrderCommand) error {
	return s.Command.CancelOrder(ctx, cmd)
}

// UpdateOrderExecution 更新订单执行状态
func (s *OrderService) UpdateOrderExecution(ctx context.Context, orderID string, filledQty, tradePrice float64) error {
	return s.Command.UpdateOrderExecution(ctx, orderID, filledQty, tradePrice)
}

// --- Query (Reads) ---

// GetOrder 获取订单详情
func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*OrderDTO, error) {
	return s.Query.GetOrder(ctx, orderID)
}

// ListOrders 列出用户订单
func (s *OrderService) ListOrders(ctx context.Context, userID string, status domain.OrderStatus, limit, offset int) ([]*OrderDTO, int64, error) {
	return s.Query.ListOrders(ctx, userID, status, limit, offset)
}

// SetDTMServer 设置 DTM 服务器地址
func (s *OrderService) SetDTMServer(server string) {
	s.DTMServer = server
}
