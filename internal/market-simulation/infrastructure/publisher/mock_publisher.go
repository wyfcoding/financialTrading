package publisher

import (
	"context"

	"github.com/wyfcoding/financialTrading/internal/market-simulation/domain"
	"github.com/wyfcoding/pkg/logging"
)

// MockMarketDataPublisher 模拟市场数据发布者
type MockMarketDataPublisher struct{}

// NewMockMarketDataPublisher 创建模拟市场数据发布者
func NewMockMarketDataPublisher() domain.MarketDataPublisher {
	return &MockMarketDataPublisher{}
}

// Publish 发布市场数据（模拟实现）
func (p *MockMarketDataPublisher) Publish(ctx context.Context, symbol string, price float64) error {
	logging.Info(ctx, "Publishing market data",
		"publisher", "MockMarketDataPublisher",
		"symbol", symbol,
		"price", price,
	)
	return nil
}
