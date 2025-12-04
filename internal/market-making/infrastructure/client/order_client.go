package client

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialTrading/go-api/order"
	"github.com/wyfcoding/financialTrading/internal/market-making/domain"
	"github.com/wyfcoding/financialTrading/pkg/grpcclient"
	"github.com/wyfcoding/financialTrading/pkg/logger"
)

// OrderClientImpl 订单服务客户端实现
type OrderClientImpl struct {
	client order.OrderServiceClient
}

// NewOrderClient 创建订单服务客户端
func NewOrderClient(cfg grpcclient.ClientConfig) (domain.OrderClient, error) {
	conn, err := grpcclient.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create order service client: %w", err)
	}

	return &OrderClientImpl{
		client: order.NewOrderServiceClient(conn),
	}, nil
}

// PlaceOrder 下单
func (c *OrderClientImpl) PlaceOrder(ctx context.Context, symbol string, side string, price, quantity float64) (string, error) {
	req := &order.CreateOrderRequest{
		Symbol:    symbol,
		Side:      side,
		Price:     fmt.Sprintf("%f", price),
		Quantity:  fmt.Sprintf("%f", quantity),
		OrderType: "LIMIT", // 默认限价单
		// TimeInForce: "GTC",
	}

	resp, err := c.client.CreateOrder(ctx, req)
	if err != nil {
		logger.Error(ctx, "Failed to place order",
			"symbol", symbol,
			"side", side,
			"price", price,
			"quantity", quantity,
			"error", err,
		)
		return "", err
	}

	return resp.OrderId, nil
}
