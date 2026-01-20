package client

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	positionv1 "github.com/wyfcoding/financialtrading/go-api/position/v1"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/grpcclient"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/metrics"
	"google.golang.org/grpc"
)

// OrderClientImpl 订单服务客户端实现
type OrderClientImpl struct {
	orderCli    orderv1.OrderServiceClient
	positionCli positionv1.PositionServiceClient
}

// NewOrderClient 创建订单服务客户端
func NewOrderClient(target string, m *metrics.Metrics, cbCfg config.CircuitBreakerConfig) (domain.OrderClient, error) {
	conn, err := grpcclient.NewClientFactory(logging.Default(), m, cbCfg).NewClient(target)
	if err != nil {
		return nil, fmt.Errorf("failed to create order service client: %w", err)
	}

	return &OrderClientImpl{
		orderCli:    orderv1.NewOrderServiceClient(conn),
		positionCli: positionv1.NewPositionServiceClient(conn),
	}, nil
}

// NewOrderClientFromConn 从现有连接创建客户端
func NewOrderClientFromConn(conn *grpc.ClientConn) domain.OrderClient {
	return &OrderClientImpl{
		orderCli:    orderv1.NewOrderServiceClient(conn),
		positionCli: positionv1.NewPositionServiceClient(conn),
	}
}

// PlaceOrder 下单
func (c *OrderClientImpl) PlaceOrder(ctx context.Context, symbol string, side string, price, quantity decimal.Decimal) (string, error) {
	req := &orderv1.CreateOrderRequest{
		Symbol:    symbol,
		Side:      side,
		Price:     price.String(),
		Quantity:  quantity.String(),
		OrderType: "LIMIT",
	}

	resp, err := c.orderCli.CreateOrder(ctx, req)
	if err != nil {
		logging.Error(ctx, "Failed to place order",
			"symbol", symbol,
			"side", side,
			"price", price,
			"quantity", quantity,
			"error", err,
		)
		return "", err
	}

	if resp.Order == nil {
		return "", fmt.Errorf("order response is nil")
	}

	return resp.Order.OrderId, nil
}

// GetPosition 获取持仓
func (c *OrderClientImpl) GetPosition(ctx context.Context, symbol string) (decimal.Decimal, error) {
	// 获取该交易对的所有持仓
	resp, err := c.positionCli.GetPositions(ctx, &positionv1.GetPositionsRequest{
		Symbol: symbol,
	})
	if err != nil {
		return decimal.Zero, err
	}

	total := decimal.Zero
	for _, p := range resp.Positions {
		qty, err := decimal.NewFromString(p.Quantity)
		if err != nil {
			continue
		}
		// 多头为正，空头为负
		if p.Side == "LONG" {
			total = total.Add(qty)
		} else {
			total = total.Sub(qty)
		}
	}

	return total, nil
}
