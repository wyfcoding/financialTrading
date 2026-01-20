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
	var orderSide orderv1.OrderSide
	switch side {
	case "BUY":
		orderSide = orderv1.OrderSide_BUY
	case "SELL":
		orderSide = orderv1.OrderSide_SELL
	}

	req := &orderv1.CreateOrderRequest{
		UserId:   "MARKET_MAKER", // Placeholder
		Symbol:   symbol,
		Side:     orderSide,
		Price:    price.InexactFloat64(),
		Quantity: quantity.InexactFloat64(),
		Type:     orderv1.OrderType_LIMIT,
	}

	resp, err := c.orderCli.CreateOrder(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.OrderId, nil
}

// GetPosition 获取持仓
func (c *OrderClientImpl) GetPosition(ctx context.Context, symbol string) (decimal.Decimal, error) {
	// 获取该用户（做市商）的所有持仓
	resp, err := c.positionCli.GetPositions(ctx, &positionv1.GetPositionsRequest{
		UserId:   "MARKET_MAKER",
		PageSize: 100,
		Page:     1,
	})
	if err != nil {
		return decimal.Zero, err
	}

	total := decimal.Zero
	for _, p := range resp.Positions {
		if p.Symbol != symbol {
			continue
		}
		qty, err := decimal.NewFromString(p.Quantity)
		if err != nil {
			continue
		}
		// 多头为正，空头为负
		if p.Side == "LONG" || p.Side == "BUY" {
			total = total.Add(qty)
		} else {
			total = total.Sub(qty)
		}
	}

	return total, nil
}
