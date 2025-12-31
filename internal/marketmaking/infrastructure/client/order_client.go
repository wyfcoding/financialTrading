package client

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	orderv1 "github.com/wyfcoding/financialtrading/goapi/order/v1"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/grpcclient"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/metrics"
	"google.golang.org/grpc"
)

// OrderClientImpl 订单服务客户端实现
type OrderClientImpl struct {
	client orderv1.OrderServiceClient
}

// NewOrderClient 创建订单服务客户端
func NewOrderClient(target string, m *metrics.Metrics, cbCfg config.CircuitBreakerConfig) (domain.OrderClient, error) {
	conn, err := grpcclient.NewClientFactory(logging.Default(), m, cbCfg).NewClient(target)
	if err != nil {
		return nil, fmt.Errorf("failed to create order service client: %w", err)
	}

	return &OrderClientImpl{
		client: orderv1.NewOrderServiceClient(conn),
	}, nil
}

// NewOrderClientFromConn 从现有连接创建客户端
func NewOrderClientFromConn(conn *grpc.ClientConn) domain.OrderClient {
	return &OrderClientImpl{
		client: orderv1.NewOrderServiceClient(conn),
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

	resp, err := c.client.CreateOrder(ctx, req)
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
