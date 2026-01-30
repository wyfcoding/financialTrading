package application

import (
	"context"
	"log/slog"

	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	positionv1 "github.com/wyfcoding/financialtrading/go-api/position/v1"
	riskv1 "github.com/wyfcoding/financialtrading/go-api/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/security/risk"
)

// OrderService 作为订单服务操作的门面。
type OrderService struct {
	Command *OrderCommandService
	Query   *OrderQueryService
	logger  *slog.Logger
}

// NewOrderService 构造函数。
func NewOrderService(repo domain.OrderRepository, riskEvaluator risk.Evaluator, logger *slog.Logger) *OrderService {
	return &OrderService{
		Command: NewOrderCommandService(repo, riskEvaluator, logger),
		Query:   NewOrderQueryService(repo),
		logger:  logger.With("module", "order_service"),
	}
}

func (s *OrderService) SetRiskClient(cli riskv1.RiskServiceClient) {
	s.Command.SetRiskClient(cli)
}

func (s *OrderService) SetAccountClient(cli accountv1.AccountServiceClient, svcURL string) {
	s.Command.SetAccountClient(cli, svcURL)
}

func (s *OrderService) SetPositionClient(cli positionv1.PositionServiceClient, svcURL string) {
	s.Command.SetPositionClient(cli, svcURL)
}

func (s *OrderService) SetDTMServer(addr string) {
	s.Command.SetDTMServer(addr)
}

// --- 写操作 (Delegates to Command) ---

func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderDTO, error) {
	return s.Command.CreateOrder(ctx, req)
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID, userID string) (*OrderDTO, error) {
	return s.Command.CancelOrder(ctx, orderID, userID)
}

func (s *OrderService) HandleTradeExecuted(ctx context.Context, event map[string]any) error {
	return s.Command.HandleTradeExecuted(ctx, event)
}

// --- 读操作 (Delegates to Query) ---

func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*OrderDTO, error) {
	return s.Query.GetOrder(ctx, orderID)
}

func (s *OrderService) ListOrders(ctx context.Context, userID string, status domain.OrderStatus, limit, offset int) ([]*OrderDTO, int64, error) {
	return s.Query.ListOrders(ctx, userID, status, limit, offset)
}
