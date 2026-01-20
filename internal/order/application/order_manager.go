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
)

// OrderManager 处理所有订单相关的写入操作（Commands）。
type OrderManager struct {
	repo           domain.OrderRepository
	riskEvaluator  risk.Evaluator
	riskCli        riskv1.RiskServiceClient
	accountCli     accountv1.AccountServiceClient
	positionCli    positionv1.PositionServiceClient
	dtmServer      string
	accountSvcURL  string // DTM 回调用的 Account 服务地址
	positionSvcURL string // DTM 回调用的 Position 服务地址
	logger         *slog.Logger
}

// NewOrderManager 构造函数。
func NewOrderManager(repo domain.OrderRepository, riskEvaluator risk.Evaluator, logger *slog.Logger) *OrderManager {
	return &OrderManager{
		repo:          repo,
		riskEvaluator: riskEvaluator,
		logger:        logger.With("module", "order_manager"),
	}
}

func (m *OrderManager) SetRiskClient(cli riskv1.RiskServiceClient) {
	m.riskCli = cli
}

func (m *OrderManager) SetAccountClient(cli accountv1.AccountServiceClient, svcURL string) {
	m.accountCli = cli
	m.accountSvcURL = svcURL
}

func (m *OrderManager) SetPositionClient(cli positionv1.PositionServiceClient, svcURL string) {
	m.positionCli = cli
	m.positionSvcURL = svcURL
}

func (m *OrderManager) SetDTMServer(addr string) {
	m.dtmServer = addr
}

// CreateOrder 创建新订单，包含风控、同步风控调用及 DTM TCC 事务控制。
func (m *OrderManager) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderDTO, error) {
	// 性能监控埋点
	defer logging.LogDuration(ctx, "order creation completed",
		"user_id", req.UserID,
		"symbol", req.Symbol,
	)()

	logging.Info(ctx, "creating new order",
		"user_id", req.UserID,
		"symbol", req.Symbol,
		"side", req.Side,
		"order_type", req.OrderType,
	)

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

	// 1. 本地规则引擎风控预检
	riskAssessment, err := m.riskEvaluator.Assess(ctx, "trade.order_create", map[string]any{
		"user_id":  req.UserID,
		"symbol":   req.Symbol,
		"side":     req.Side,
		"amount":   totalAmount.InexactFloat64(),
		"quantity": quantity.InexactFloat64(),
	})
	if err != nil {
		logging.Error(ctx, "local risk assessment failed", "error", err)
		return nil, fmt.Errorf("security system offline")
	}

	if riskAssessment.Level == risk.Reject {
		return nil, fmt.Errorf("transaction blocked by local risk: %s", riskAssessment.Reason)
	}

	// 2. 远程金融级风控服务同步校验
	if m.riskCli != nil {
		remoteResp, err := m.riskCli.CheckRisk(ctx, &riskv1.CheckRiskRequest{
			UserId:   req.UserID,
			Symbol:   req.Symbol,
			Side:     req.Side,
			Quantity: quantity.InexactFloat64(),
			Price:    price.InexactFloat64(),
		})
		if err != nil {
			logging.Error(ctx, "remote risk assessment failed", "error", err)
			return nil, fmt.Errorf("remote risk check failed: %w", err)
		}
		if !remoteResp.Passed {
			return nil, fmt.Errorf("transaction blocked by remote risk: %s", remoteResp.Reason)
		}
		logging.Info(ctx, "remote risk check passed")
	}

	orderID := fmt.Sprintf("ORD-%d", idgen.GenID())
	order := domain.NewOrder(
		orderID,
		req.UserID,
		req.Symbol,
		domain.OrderSide(req.Side),
		domain.OrderType(req.OrderType),
		price.InexactFloat64(),
		quantity.InexactFloat64(),
	)

	// --- 3. 开启分布式 TCC 事务控制资产冻结 ---
	if m.dtmServer != "" {
		logging.Info(ctx, "initiating tcc transaction for order", "order_id", orderID, "side", order.Side)

		gid := orderID
		tcc := dtm.NewTcc(m.dtmServer, gid)

		err = tcc.Execute(ctx, func(t *dtmgrpc.TccGrpc) error {
			switch order.Side {
			case domain.SideBuy:
				// 买单：冻结账户 USDT 余额
				accountGrpcPrefix := m.accountSvcURL + "/api.account.v1.AccountService"
				freezeReq := &accountv1.TccFreezeRequest{
					UserId:   req.UserID,
					Currency: "USDT",
					Amount:   totalAmount.String(),
					OrderId:  orderID,
				}
				if err := dtm.CallBranch(t, freezeReq, accountGrpcPrefix+"/TccTryFreeze", accountGrpcPrefix+"/TccConfirmFreeze", accountGrpcPrefix+"/TccCancelFreeze"); err != nil {
					return err
				}
			case domain.SideSell:
				// 卖单：冻结标的资产持仓
				positionGrpcPrefix := m.positionSvcURL + "/api.position.v1.PositionService"
				freezeReq := &positionv1.TccPositionRequest{
					UserId:   req.UserID,
					Symbol:   order.Symbol,
					Quantity: decimal.NewFromFloat(order.Quantity).String(),
					OrderId:  orderID,
				}
				if err := dtm.CallBranch(t, freezeReq, positionGrpcPrefix+"/TccTryFreeze", positionGrpcPrefix+"/TccConfirmFreeze", positionGrpcPrefix+"/TccCancelFreeze"); err != nil {
					return err
				}
			}

			// 事务分支内执行订单本地持久化
			if err := m.repo.Save(ctx, order); err != nil {
				return fmt.Errorf("failed to save order locally: %w", err)
			}
			return nil
		})
		if err != nil {
			logging.Error(ctx, "tcc transaction failed", "order_id", orderID, "error", err)
			return nil, fmt.Errorf("order placement failed during transaction: %w", err)
		}
	} else {
		if err := m.repo.Save(ctx, order); err != nil {
			return nil, fmt.Errorf("failed to save order: %w", err)
		}
	}

	m.logger.InfoContext(ctx, "order created successfully", "order_id", orderID, "user_id", req.UserID)

	return &OrderDTO{
		OrderID:        order.ID, // Fix: Use ID
		UserID:         order.UserID,
		Symbol:         order.Symbol,
		Side:           string(order.Side),
		OrderType:      string(order.Type),
		Price:          decimal.NewFromFloat(order.Price).String(),
		Quantity:       decimal.NewFromFloat(order.Quantity).String(),
		FilledQuantity: decimal.NewFromFloat(order.FilledQuantity).String(),
		Status:         string(order.Status),
		CreatedAt:      order.CreatedAt.Unix(),
		UpdatedAt:      order.UpdatedAt.Unix(),
	}, nil
}

// HandleTradeExecuted 处理撮合引擎发出的成交事件。
func (m *OrderManager) HandleTradeExecuted(ctx context.Context, event map[string]any) error {
	tradeID := event["trade_id"].(string)
	buyOrderID := event["buy_order_id"].(string)
	sellOrderID := event["sell_order_id"].(string)
	quantity, _ := decimal.NewFromString(event["quantity"].(string))
	price, _ := decimal.NewFromString(event["price"].(string))

	m.logger.InfoContext(ctx, "handling trade executed event", "trade_id", tradeID, "buy_order", buyOrderID, "sell_order", sellOrderID)

	// Since we don't have updateFillStatus implemented as a separate method anymore or it was private, I will inline logic or restore it if I want to match previous behavior.
	// But to save tokens/complexity I will implement minimal update logic here.
	// Actually I should implement updateFillStatus helper.
	if err := m.updateFillStatus(ctx, buyOrderID, quantity, price, "buy match"); err != nil {
		return err
	}

	if err := m.updateFillStatus(ctx, sellOrderID, quantity, price, "sell match"); err != nil {
		return err
	}

	return nil
}

func (m *OrderManager) updateFillStatus(ctx context.Context, orderID string, qty, price decimal.Decimal, msg string) error {
	order, err := m.repo.Get(ctx, orderID)
	if err != nil || order == nil {
		return err
	}

	order.UpdateExecution(qty.InexactFloat64(), price.InexactFloat64())
	// order.Remark = ... (Order struct doesn't have Remark in domain/order.go step 2082) So skip Remark.

	if err := m.repo.Save(ctx, order); err != nil {
		m.logger.ErrorContext(ctx, "failed to update order fill status", "order_id", orderID, "error", err)
		return err
	}

	m.logger.InfoContext(ctx, "order fill status updated", "order_id", orderID, "status", order.Status)
	return nil
}

// CancelOrder 执行撤单逻辑，包含跨服务的资产解冻。
func (m *OrderManager) CancelOrder(ctx context.Context, orderID, userID string) (*OrderDTO, error) {
	order, err := m.repo.Get(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}
	if order.UserID != userID {
		return nil, fmt.Errorf("order does not belong to user: %s", userID)
	}

	if order.Status == domain.StatusFilled || order.Status == domain.StatusCancelled {
		return nil, fmt.Errorf("order cannot be cancelled, current status: %s", order.Status)
	}

	if order.Status == domain.StatusPending || order.Status == domain.StatusPartiallyFilled {
		remainingQty := decimal.NewFromFloat(order.Quantity - order.FilledQuantity)
		price := decimal.NewFromFloat(order.Price)

		switch order.Side {
		case domain.SideBuy:
			remainingAmount := remainingQty.Mul(price)
			if m.accountCli != nil {
				_, err := m.accountCli.TccCancelFreeze(ctx, &accountv1.TccFreezeRequest{
					UserId:   userID,
					Currency: "USDT",
					Amount:   remainingAmount.String(),
					OrderId:  orderID,
				})
				if err != nil {
					logging.Error(ctx, "failed to unfreeze funds during cancellation", "order_id", orderID, "error", err)
					return nil, fmt.Errorf("failed to unfreeze funds: %w", err)
				}
			}
		case domain.SideSell:
			if m.positionCli != nil {
				_, err := m.positionCli.TccCancelFreeze(ctx, &positionv1.TccPositionRequest{
					UserId:   userID,
					Symbol:   order.Symbol,
					Quantity: remainingQty.String(),
					OrderId:  orderID,
				})
				if err != nil {
					logging.Error(ctx, "failed to unfreeze assets during cancellation", "order_id", orderID, "error", err)
					return nil, fmt.Errorf("failed to unfreeze assets: %w", err)
				}
			}
		}
	}

	if err := m.repo.UpdateStatus(ctx, orderID, domain.StatusCancelled); err != nil {
		logging.Error(ctx, "failed to update order status after unfreezing", "order_id", orderID, "error", err)
		return nil, fmt.Errorf("failed to finalize cancellation: %w", err)
	}

	m.logger.InfoContext(ctx, "order cancelled successfully", "order_id", orderID, "user_id", userID)
	order.Status = domain.StatusCancelled

	return &OrderDTO{
		OrderID:        order.ID, // Fix: Use ID
		UserID:         order.UserID,
		Symbol:         order.Symbol,
		Side:           string(order.Side),
		OrderType:      string(order.Type),
		Price:          decimal.NewFromFloat(order.Price).String(),
		Quantity:       decimal.NewFromFloat(order.Quantity).String(),
		FilledQuantity: decimal.NewFromFloat(order.FilledQuantity).String(),
		Status:         string(order.Status),
		CreatedAt:      order.CreatedAt.Unix(),
		UpdatedAt:      order.UpdatedAt.Unix(),
	}, nil
}
