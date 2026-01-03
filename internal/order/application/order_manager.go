package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/dtm-labs/client/dtmgrpc"
	accountv1 "github.com/wyfcoding/financialtrading/goapi/account/v1"
	riskv1 "github.com/wyfcoding/financialtrading/goapi/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/dtm"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/security/risk"
)

// OrderManager 处理所有订单相关的写入操作（Commands）。
type OrderManager struct {
	repo          domain.OrderRepository
	riskEvaluator risk.Evaluator
	riskCli       riskv1.RiskServiceClient
	accountCli    accountv1.AccountServiceClient
	dtmServer     string
	accountSvcURL string // DTM 回调用的 Account 服务地址 (例如 "account:50051")
}

// NewOrderManager 构造函数。
func NewOrderManager(repo domain.OrderRepository, riskEvaluator risk.Evaluator) *OrderManager {
	return &OrderManager{
		repo:          repo,
		riskEvaluator: riskEvaluator,
	}
}

func (m *OrderManager) SetRiskClient(cli riskv1.RiskServiceClient) {
	m.riskCli = cli
}

func (m *OrderManager) SetAccountClient(cli accountv1.AccountServiceClient, svcURL string) {
	m.accountCli = cli
	m.accountSvcURL = svcURL
}

func (m *OrderManager) SetDTMServer(addr string) {
	m.dtmServer = addr
}

// CreateOrder 创建订单
func (m *OrderManager) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderDTO, error) {
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

	totalAmount := price.Mul(quantity)

	// 1. 本地风控评估 (快速检查)
	riskAssessment, err := m.riskEvaluator.Assess(ctx, "trade.order_create", map[string]any{
		"user_id":  req.UserID,
		"symbol":   req.Symbol,
		"side":     req.Side,
		"amount":   totalAmount.InexactFloat64(),
		"quantity": quantity.InexactFloat64(),
	})
	if err != nil {
		logging.Error(ctx, "Local risk assessment failed", "error", err)
		return nil, fmt.Errorf("security system offline")
	}

	if riskAssessment.Level == risk.Reject {
		return nil, fmt.Errorf("transaction blocked by local risk: %s", riskAssessment.Reason)
	}

	// 2. 远程风控评估 (gRPC 同步调用 - Internal Interaction)
	if m.riskCli != nil {
		remoteResp, err := m.riskCli.AssessRisk(ctx, &riskv1.AssessRiskRequest{
			UserId:   req.UserID,
			Symbol:   req.Symbol,
			Side:     req.Side,
			Quantity: quantity.String(),
			Price:    price.String(),
		})
		if err != nil {
			logging.Error(ctx, "Remote risk assessment failed", "error", err)
			return nil, fmt.Errorf("remote risk check failed: %w", err)
		}
		if !remoteResp.IsAllowed {
			return nil, fmt.Errorf("transaction blocked by remote risk: %s (level: %s)", remoteResp.Reason, remoteResp.RiskLevel)
		}
		logging.Info(ctx, "Remote risk check passed", "risk_level", remoteResp.RiskLevel, "score", remoteResp.RiskScore)
	}

	// 生成订单 ID
	orderID := fmt.Sprintf("ORD-%d", idgen.GenID())

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

	// --- 3. DTM TCC 分布式事务 ---
	// 如果是买单，需要冻结资金。卖单需要冻结持仓（此处简化，假设只处理买单资金冻结，或者卖单需要 PositionService）
	// 为演示目的，假设买单 (Side=BUY) 冻结 USDT。
	if order.Side == domain.OrderSideBuy && m.dtmServer != "" {
		logging.Info(ctx, "Initiating TCC transaction for BUY order", "order_id", orderID)

		// 创建 TCC 事务
		gid := orderID // 使用订单ID作为全局事务ID
		tcc := dtm.NewTcc(ctx, m.dtmServer, gid)

		err = tcc.Execute(func(t *dtmgrpc.TccGrpc) error {
			// Step 3.1: 资金冻结 (Try)
			// 注意：AccountService 的地址通常需要配置。这里假设通过 accountSvcURL 传入。
			// TCC 的 CallBranch 需要传入 Try, Confirm, Cancel 的完整 URL 或 gRPC Method
			accountGrpcPrefix := m.accountSvcURL + "/api.account.v1.AccountService"

			freezeReq := &accountv1.TccFreezeRequest{
				UserId:   req.UserID,
				Currency: "USDT", // 假设计价货币是 USDT，实际应从 Symbol 获取 (e.g., BTC/USDT)
				Amount:   totalAmount.String(),
				OrderId:  orderID,
			}

			// 注册分支事务: Account.TccFreeze
			if err := dtm.CallBranch(
				t,
				freezeReq,
				accountGrpcPrefix+"/TccTryFreeze",
				accountGrpcPrefix+"/TccConfirmFreeze",
				accountGrpcPrefix+"/TccCancelFreeze",
			); err != nil {
				return fmt.Errorf("failed to call TccFreeze: %w", err)
			}

			// Step 3.2: 订单创建 (Try)
			// 实际上订单创建本身也可以作为 TCC 的一部分，或者在 TCC Execute 内部执行本地事务。
			// 在这里，我们将“保存订单”视为 TCC 的“Confirm”逻辑（或者 TCC 主体逻辑）。
			// 但因为我们已经在 OrderService 内部，我们可以直接执行 DB 操作。
			// 如果 DB 操作失败，TCC 会自动回滚（调用 Account 的 Cancel）。

			if err := m.repo.Save(ctx, order); err != nil {
				return fmt.Errorf("failed to save order locally: %w", err)
			}
			return nil
		})

		if err != nil {
			logging.Error(ctx, "TCC transaction failed", "order_id", orderID, "error", err)
			return nil, fmt.Errorf("order placement failed during transaction: %w", err)
		}

	} else {
		// 无 DTM 或卖单（暂未实现卖单 TCC），走普通逻辑
		if err := m.repo.Save(ctx, order); err != nil {
			return nil, fmt.Errorf("failed to save order: %w", err)
		}
	}

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
	if !order.CanBeCancelled() {
		return nil, fmt.Errorf("order cannot be cancelled, current status: %s", order.Status)
	}

	if err := m.repo.UpdateStatus(ctx, orderID, domain.OrderStatusCancelled); err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

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
