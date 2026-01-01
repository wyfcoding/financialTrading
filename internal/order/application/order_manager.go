package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	riskv1 "github.com/wyfcoding/financialtrading/goapi/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/security/risk"
)

// OrderManager 处理所有订单相关的写入操作（Commands）。
type OrderManager struct {
	repo          domain.OrderRepository
	riskEvaluator risk.Evaluator
	riskCli       riskv1.RiskServiceClient
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

	// 1. 本地风控评估 (快速检查)
	riskAssessment, err := m.riskEvaluator.Assess(ctx, "trade.order_create", map[string]any{
		"user_id":  req.UserID,
		"symbol":   req.Symbol,
		"side":     req.Side,
		"amount":   price.Mul(quantity).InexactFloat64(),
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
			// 为了系统可用性，这里可以决定是否降级。由于是金融交易，优先报错。
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

	// 保存到仓储
	if err := m.repo.Save(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
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
