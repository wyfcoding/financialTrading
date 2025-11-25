package infrastructure

import (
	"context"
	"math/rand"
	"time"

	"github.com/wyfcoding/financialTrading/internal/quant/domain"
)

// MockMarketDataClient 模拟市场数据客户端
type MockMarketDataClient struct{}

func NewMockMarketDataClient() domain.MarketDataClient {
	return &MockMarketDataClient{}
}

func (c *MockMarketDataClient) GetHistoricalData(ctx context.Context, symbol string, start, end time.Time) ([]float64, error) {
	// 模拟生成一些随机价格数据
	days := int(end.Sub(start).Hours() / 24)
	if days <= 0 {
		days = 10
	}

	prices := make([]float64, days)
	basePrice := 100.0
	for i := 0; i < days; i++ {
		change := (rand.Float64() - 0.5) * 2 // -1 to 1
		basePrice += change
		if basePrice < 0 {
			basePrice = 0.1
		}
		prices[i] = basePrice
	}

	return prices, nil
}
