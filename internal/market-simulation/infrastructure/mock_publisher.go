package infrastructure

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialTrading/internal/market-simulation/domain"
)

// MockMarketDataPublisher 模拟市场数据发布者
type MockMarketDataPublisher struct{}

// NewMockMarketDataPublisher 创建模拟市场数据发布者
func NewMockMarketDataPublisher() domain.MarketDataPublisher {
	return &MockMarketDataPublisher{}
}

// Publish 发布市场数据
func (p *MockMarketDataPublisher) Publish(ctx context.Context, symbol string, price float64) error {
	fmt.Printf("[MockMarketDataPublisher] Publishing price for %s: %f\n", symbol, price)
	return nil
}
