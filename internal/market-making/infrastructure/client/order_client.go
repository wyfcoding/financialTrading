package client

import (
	"context"
	"fmt"

	orderv1 "github.com/wyfcoding/financialTrading/go-api/order/v1"
	"github.com/wyfcoding/financialTrading/internal/market-making/domain"
	"github.com/wyfcoding/pkg/grpcclient"
	"github.com/wyfcoding/pkg/logging"
	"google.golang.org/grpc"
)

// OrderClientImpl 订单服务客户端实现
type OrderClientImpl struct {
	client orderv1.OrderServiceClient
}

// NewOrderClient 创建订单服务客户端
func NewOrderClient(cfg grpcclient.ClientConfig) (domain.OrderClient, error) {
	conn, err := grpcclient.NewClient(cfg)
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
func (c *OrderClientImpl) PlaceOrder(ctx context.Context, symbol string, side string, price, quantity float64) (string, error) {
	req := &orderv1.CreateOrderRequest{
		Symbol:    symbol,
		Side:      side,
		Price:     fmt.Sprintf("%f", price),
		Quantity:  fmt.Sprintf("%f", quantity),
		OrderType: "LIMIT", // 默认限价单
		// TimeInForce: "GTC",
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