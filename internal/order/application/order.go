package application

import (
	"context"

	riskv1 "github.com/wyfcoding/financialtrading/goapi/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/security/risk"
)

// OrderService 订单门面服务，整合 Manager 和 Query。
type OrderService struct {
	manager *OrderManager
	query   *OrderQuery
}

// NewOrderService 构造函数。
func NewOrderService(repo domain.OrderRepository, riskEvaluator risk.Evaluator) *OrderService {
	return &OrderService{
		manager: NewOrderManager(repo, riskEvaluator),
		query:   NewOrderQuery(repo),
	}
}

func (s *OrderService) SetRiskClient(cli riskv1.RiskServiceClient) {
	s.manager.SetRiskClient(cli)
}

// --- Manager (Writes) ---

func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderDTO, error) {
	return s.manager.CreateOrder(ctx, req)
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID, userID string) (*OrderDTO, error) {
	return s.manager.CancelOrder(ctx, orderID, userID)
}

// --- Query (Reads) ---

func (s *OrderService) GetOrder(ctx context.Context, orderID, userID string) (*OrderDTO, error) {
	return s.query.GetOrder(ctx, orderID, userID)
}

func (s *OrderService) ListOrders(ctx context.Context, userID string, status domain.OrderStatus, limit, offset int) ([]*OrderDTO, int64, error) {
	return s.query.ListOrders(ctx, userID, status, limit, offset)
}

// --- Legacy Compatibility Types ---

// CreateOrderRequest 创建订单请求 DTO
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
