package infrastructure

import (
	"context"
	"fmt"

	"github.com/fynnwu/FinancialTrading/internal/market-simulation/domain"
)

// MockMarketDataPublisher 模拟市场数据发布者
type MockMarketDataPublisher struct{}

func NewMockMarketDataPublisher() domain.MarketDataPublisher {
	return &MockMarketDataPublisher{}
}

func (p *MockMarketDataPublisher) Publish(ctx context.Context, symbol string, price float64) error {
	fmt.Printf("[MockMarketDataPublisher] Publishing price for %s: %f\n", symbol, price)
	return nil
}
