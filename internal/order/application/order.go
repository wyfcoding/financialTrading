package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/financialtrading/internal/order/infrastructure/messaging"
	"github.com/wyfcoding/financialtrading/internal/order/infrastructure/persistence/mysql"
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
	Command   *OrderCommandService
	Query     *OrderQueryService
	DTMServer string
}

// NewOrderService 构造函数
func NewOrderService(repo domain.OrderRepository, searchRepo domain.OrderSearchRepository, db *gorm.DB) (*OrderService, error) {
	// 创建事件存储
	eventStore := mysql.NewEventStore(db)

	// 创建事件发布者
	eventPublisher := messaging.NewOutboxEventPublisher(db)

	// 创建命令服务
	command := NewOrderCommandService(repo, eventStore, eventPublisher, db)

	// 创建查询服务
	query := NewOrderQueryService(repo, searchRepo)

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

// --- DTO Definitions ---

type CreateOrderRequest struct {
	UserID        string
	Symbol        string
	Side          string
	OrderType     string
	Price         string
	Quantity      string
	TimeInForce   string
	StopPrice     string
	ClientOrderID string

	// Bracket support
	TakeProfitPrice string
	StopLossPrice   string

	// OCO support
	IsOCO         bool
	LinkedOrderID string
}

type OrderDTO struct {
	OrderID        string `json:"order_id"`
	UserID         string `json:"user_id"`
	Symbol         string `json:"symbol"`
	Side           string `json:"side"`
	OrderType      string `json:"order_type"`
	Price          string `json:"price"`
	Quantity       string `json:"quantity"`
	FilledQuantity string `json:"filled_quantity"`
	AveragePrice   string `json:"average_price"`
	Status         string `json:"status"`
	TimeInForce    string `json:"time_in_force"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
	Remark         string `json:"remark"`
}
