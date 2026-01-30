package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dtm-labs/client/dtmgrpc"
	"github.com/shopspring/decimal"
	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	positionv1 "github.com/wyfcoding/financialtrading/go-api/position/v1"
	riskv1 "github.com/wyfcoding/financialtrading/go-api/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/dtm"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/security/risk"
	"github.com/wyfcoding/pkg/tracing"
	"github.com/wyfcoding/pkg/transaction"
	"go.opentelemetry.io/otel/trace"
)

// OrderCommandService 处理所有订单相关的写入操作（Commands）。
type OrderCommandService struct {
	repo           domain.OrderRepository
	riskEvaluator  risk.Evaluator
	riskCli        riskv1.RiskServiceClient
	accountCli     accountv1.AccountServiceClient
	positionCli    positionv1.PositionServiceClient
	dtmServer      string
	accountSvcURL  string
	positionSvcURL string
	logger         *slog.Logger
}

// NewOrderCommandService 构造函数。
func NewOrderCommandService(repo domain.OrderRepository, riskEvaluator risk.Evaluator, logger *slog.Logger) *OrderCommandService {
	return &OrderCommandService{
		repo:          repo,
		riskEvaluator: riskEvaluator,
		logger:        logger.With("module", "order_command"),
	}
}

func (s *OrderCommandService) SetRiskClient(cli riskv1.RiskServiceClient) {
	s.riskCli = cli
}

func (s *OrderCommandService) SetAccountClient(cli accountv1.AccountServiceClient, svcURL string) {
	s.accountCli = cli
	s.accountSvcURL = svcURL
}

func (s *OrderCommandService) SetPositionClient(cli positionv1.PositionServiceClient, svcURL string) {
	s.positionCli = cli
	s.positionSvcURL = svcURL
}

func (s *OrderCommandService) SetDTMServer(addr string) {
	s.dtmServer = addr
}

func (s *OrderCommandService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderDTO, error) {
	ctx, span := tracing.Tracer().Start(ctx, "OrderCommandService.CreateOrder", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	defer logging.LogDuration(ctx, "order creation completed",
		"user_id", req.UserID,
		"symbol", req.Symbol,
	)()

	if req.UserID == "" || req.Symbol == "" || req.Side == "" {
		return nil, fmt.Errorf("invalid request parameters")
	}

	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}
	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	totalAmount := price.Mul(quantity)

	// 1. Local Risk Check
	riskAssessment, err := s.riskEvaluator.Assess(ctx, "trade.order_create", map[string]any{
		"user_id":  req.UserID,
		"symbol":   req.Symbol,
		"side":     req.Side,
		"amount":   totalAmount.InexactFloat64(),
		"quantity": quantity.InexactFloat64(),
	})
	if err != nil {
		return nil, fmt.Errorf("security system offline")
	}
	if riskAssessment.Level == risk.Reject {
		return nil, fmt.Errorf("transaction blocked by local risk: %s", riskAssessment.Reason)
	}

	// 2. Remote Risk Check
	if s.riskCli != nil {
		resp, err := s.riskCli.CheckRisk(ctx, &riskv1.CheckRiskRequest{
			UserId:   req.UserID,
			Symbol:   req.Symbol,
			Side:     req.Side,
			Quantity: quantity.InexactFloat64(),
			Price:    price.InexactFloat64(),
		})
		if err != nil {
			return nil, fmt.Errorf("remote risk check failed: %w", err)
		}
		if !resp.Passed {
			return nil, fmt.Errorf("transaction blocked by remote risk: %s", resp.Reason)
		}
	}

	orderID := fmt.Sprintf("ORD-%d", idgen.GenID())
	order := domain.NewOrder(orderID, req.UserID, req.Symbol, domain.OrderSide(req.Side), domain.OrderType(req.OrderType), price.InexactFloat64(), quantity.InexactFloat64())
	order.TimeInForce = domain.TimeInForce(req.TimeInForce)
	if sp, err := decimal.NewFromString(req.StopPrice); err == nil {
		order.StopPrice = sp.InexactFloat64()
	}
	order.IsOCO = req.IsOCO
	order.ParentOrderID = req.LinkedOrderID

	// 3. TCC Transaction
	if s.dtmServer != "" {
		tcc := dtm.NewTcc(s.dtmServer, orderID)
		err = tcc.Execute(ctx, func(t *dtmgrpc.TccGrpc) error {
			switch order.Side {
			case domain.SideBuy:
				grpcURL := s.accountSvcURL + "/api.account.v1.AccountService"
				freezeReq := &accountv1.TccFreezeRequest{
					UserId:   req.UserID,
					Currency: "USDT",
					Amount:   totalAmount.String(),
					OrderId:  orderID,
				}
				if err := dtm.CallBranch(t, freezeReq, grpcURL+"/TccTryFreeze", grpcURL+"/TccConfirmFreeze", grpcURL+"/TccCancelFreeze"); err != nil {
					return err
				}
			case domain.SideSell:
				grpcURL := s.positionSvcURL + "/api.position.v1.PositionService"
				freezeReq := &positionv1.TccPositionRequest{
					UserId:   req.UserID,
					Symbol:   order.Symbol,
					Quantity: quantity.String(),
					OrderId:  orderID,
				}
				if err := dtm.CallBranch(t, freezeReq, grpcURL+"/TccTryFreeze", grpcURL+"/TccConfirmFreeze", grpcURL+"/TccCancelFreeze"); err != nil {
					return err
				}
			}

			if err := s.repo.Save(ctx, order); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		if err := s.repo.Save(ctx, order); err != nil {
			return nil, err
		}
	}

	return s.toDTO(order), nil
}

func (s *OrderCommandService) CancelOrder(ctx context.Context, orderID, userID string) (*OrderDTO, error) {
	order, err := s.repo.Get(ctx, orderID)
	if err != nil || order == nil {
		return nil, fmt.Errorf("order not found")
	}
	if order.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	if order.Status == domain.StatusFilled || order.Status == domain.StatusCancelled {
		return nil, fmt.Errorf("cannot cancel order in state %s", order.Status)
	}

	remainingQty := order.Quantity - order.FilledQuantity
	if remainingQty > 0 {
		switch order.Side {
		case domain.SideBuy:
			if s.accountCli != nil {
				amount := decimal.NewFromFloat(remainingQty).Mul(decimal.NewFromFloat(order.Price)).String()
				_, err = s.accountCli.TccCancelFreeze(ctx, &accountv1.TccFreezeRequest{
					UserId:   userID,
					Currency: "USDT",
					Amount:   amount,
					OrderId:  orderID,
				})
				if err != nil {
					return nil, err
				}
			}
		case domain.SideSell:
			if s.positionCli != nil {
				_, err = s.positionCli.TccCancelFreeze(ctx, &positionv1.TccPositionRequest{
					UserId:   userID,
					Symbol:   order.Symbol,
					Quantity: decimal.NewFromFloat(remainingQty).String(),
					OrderId:  orderID,
				})
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if err := s.repo.UpdateStatus(ctx, orderID, domain.StatusCancelled); err != nil {
		return nil, err
	}
	order.Status = domain.StatusCancelled
	return s.toDTO(order), nil
}

func (s *OrderCommandService) HandleTradeExecuted(ctx context.Context, event map[string]any) error {
	buyOrderID := event["buy_order_id"].(string)
	sellOrderID := event["sell_order_id"].(string)
	qty, _ := decimal.NewFromString(event["quantity"].(string))
	price, _ := decimal.NewFromString(event["price"].(string))

	if err := s.updateFillStatus(ctx, buyOrderID, qty.InexactFloat64(), price.InexactFloat64()); err != nil {
		return err
	}
	if err := s.updateFillStatus(ctx, sellOrderID, qty.InexactFloat64(), price.InexactFloat64()); err != nil {
		return err
	}
	return nil
}

func (s *OrderCommandService) updateFillStatus(ctx context.Context, orderID string, qty, price float64) error {
	order, err := s.repo.Get(ctx, orderID)
	if err != nil || order == nil {
		return err
	}

	order.UpdateExecution(qty, price)
	if err := s.repo.Save(ctx, order); err != nil {
		return err
	}

	if order.IsOCO && order.ParentOrderID != "" && (order.Status == domain.StatusFilled || order.Status == domain.StatusCancelled) {
		go s.CancelOrder(context.Background(), order.ParentOrderID, order.UserID)
	}
	return nil
}

func (s *OrderCommandService) toDTO(o *domain.Order) *OrderDTO {
	return &OrderDTO{
		OrderID:        o.OrderID,
		UserID:         o.UserID,
		Symbol:         o.Symbol,
		Side:           string(o.Side),
		OrderType:      string(o.Type),
		Price:          decimal.NewFromFloat(o.Price).String(),
		Quantity:       decimal.NewFromFloat(o.Quantity).String(),
		FilledQuantity: decimal.NewFromFloat(o.FilledQuantity).String(),
		Status:         string(o.Status),
		TimeInForce:    string(o.TimeInForce),
		CreatedAt:      o.CreatedAt.Unix(),
		UpdatedAt:      o.UpdatedAt.Unix(),
	}
}

// --- Saga Steps ---

type OrderCreateStep struct {
	transaction.BaseStep
	repo  domain.OrderRepository
	order *domain.Order
}

func (s *OrderCreateStep) Execute(ctx context.Context) error {
	s.order.Status = domain.StatusPending
	return s.repo.Save(ctx, s.order)
}

func (s *OrderCreateStep) Compensate(ctx context.Context) error {
	return s.repo.UpdateStatus(ctx, s.order.OrderID, domain.StatusCancelled)
}

type RiskCheckStep struct {
	transaction.BaseStep
	riskCli riskv1.RiskServiceClient
	order   *domain.Order
}

func (s *RiskCheckStep) Execute(ctx context.Context) error {
	if s.riskCli == nil {
		return nil
	}
	resp, err := s.riskCli.CheckRisk(ctx, &riskv1.CheckRiskRequest{
		UserId:   s.order.UserID,
		Symbol:   s.order.Symbol,
		Side:     string(s.order.Side),
		Quantity: s.order.Quantity,
		Price:    s.order.Price,
	})
	if err != nil || !resp.Passed {
		return fmt.Errorf("risk check failed")
	}
	return nil
}

func (s *RiskCheckStep) Compensate(ctx context.Context) error {
	return nil
}

type AccountBalanceStep struct {
	transaction.BaseStep
	accountCli accountv1.AccountServiceClient
	order      *domain.Order
	amount     string
}

func (s *AccountBalanceStep) Execute(ctx context.Context) error {
	if s.accountCli == nil {
		return nil
	}
	return nil // Mocked
}

func (s *AccountBalanceStep) Compensate(ctx context.Context) error {
	return nil
}
