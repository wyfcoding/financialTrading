package infrastructure

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/wyfcoding/financialTrading/internal/pricing/domain"
)

// MockMarketDataClient 模拟市场数据客户端
type MockMarketDataClient struct{}

func NewMockMarketDataClient() domain.MarketDataClient {
	return &MockMarketDataClient{}
}

func (c *MockMarketDataClient) GetPrice(ctx context.Context, symbol string) (float64, error) {
	// 模拟生成随机价格
	price := 100.0 + (rand.Float64()-0.5)*10
	fmt.Printf("[MockMarketDataClient] Got price for %s: %f\n", symbol, price)
	return price, nil
}
