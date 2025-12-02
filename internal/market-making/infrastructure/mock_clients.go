package infrastructure

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/wyfcoding/financialTrading/internal/market-making/domain"
)

// MockOrderClient 模拟订单服务客户端
type MockOrderClient struct{}

// NewMockOrderClient 创建模拟订单服务客户端
func NewMockOrderClient() domain.OrderClient {
	return &MockOrderClient{}
}

// PlaceOrder 下单
func (c *MockOrderClient) PlaceOrder(ctx context.Context, symbol string, side string, price, quantity float64) (string, error) {
	fmt.Printf("[MockOrderClient] Placing order: %s %s %.2f @ %.2f\n", side, symbol, quantity, price)
	return "mock-order-id", nil
}

// MockMarketDataClient 模拟市场数据客户端
type MockMarketDataClient struct{}

// NewMockMarketDataClient 创建模拟市场数据客户端
func NewMockMarketDataClient() domain.MarketDataClient {
	return &MockMarketDataClient{}
}

// GetPrice 获取价格
func (c *MockMarketDataClient) GetPrice(ctx context.Context, symbol string) (float64, error) {
	price := 100.0 + (rand.Float64()-0.5)*10
	return price, nil
}
