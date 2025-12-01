// Package infrastructure 包含基础设施层实现
package infrastructure

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/wyfcoding/financialTrading/internal/pricing/domain"
)

// MockMarketDataClient 模拟市场数据客户端
// 用于开发和测试环境，模拟生成市场价格数据
type MockMarketDataClient struct{}

// NewMockMarketDataClient 创建模拟市场数据客户端实例
func NewMockMarketDataClient() domain.MarketDataClient {
	return &MockMarketDataClient{}
}

// GetPrice 获取最新价格
// 模拟返回一个随机价格
func (c *MockMarketDataClient) GetPrice(ctx context.Context, symbol string) (float64, error) {
	// 模拟生成随机价格 (100 +/- 5)
	price := 100.0 + (rand.Float64()-0.5)*10
	fmt.Printf("[MockMarketDataClient] Got price for %s: %f\n", symbol, price)
	return price, nil
}
