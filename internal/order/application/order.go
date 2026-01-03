package application

import (
	"context"
	"log/slog"

	accountv1 "github.com/wyfcoding/financialtrading/goapi/account/v1"
	positionv1 "github.com/wyfcoding/financialtrading/goapi/position/v1"
	riskv1 "github.com/wyfcoding/financialtrading/goapi/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/security/risk"
)

// OrderService 订单门面服务，整合 Manager 和 Query。
type OrderService struct {
	manager *OrderManager
	query   *OrderQuery
}

// NewOrderService 创建订单服务实例。
func NewOrderService(repo domain.OrderRepository, riskEvaluator risk.Evaluator, logger *slog.Logger) *OrderService {
	return &OrderService{
		manager: NewOrderManager(repo, riskEvaluator, logger),
		query:   NewOrderQuery(repo),
	}
}

func (s *OrderService) SetRiskClient(cli riskv1.RiskServiceClient) {
	s.manager.SetRiskClient(cli)
}

func (s *OrderService) SetAccountClient(cli accountv1.AccountServiceClient, svcURL string) {
	s.manager.SetAccountClient(cli, svcURL)
}

func (s *OrderService) SetPositionClient(cli positionv1.PositionServiceClient, svcURL string) {
	s.manager.SetPositionClient(cli, svcURL)
}

func (s *OrderService) SetDTMServer(addr string) {
	s.manager.SetDTMServer(addr)
}

// --- Manager (Writes) ---

func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderDTO, error) {
	return s.manager.CreateOrder(ctx, req)
}

func (s *OrderService) HandleTradeExecuted(ctx context.Context, event map[string]any) error {
	return s.manager.HandleTradeExecuted(ctx, event)
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID, userID string) (*OrderDTO, error) {
	return s.manager.CancelOrder(ctx, orderID, userID)
}

// --- Query (Reads) ---

func (s *OrderService) GetOrder(ctx context.Context, orderID, userID string) (*OrderDTO, error) {
	return s.query.GetOrder(ctx, orderID, userID)
}

func (s *OrderService) ListOrders(ctx context.Context, userID, symbol string, status domain.OrderStatus, limit, offset int) ([]*OrderDTO, int64, error) {
	return s.query.ListOrders(ctx, userID, symbol, status, limit, offset)
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
