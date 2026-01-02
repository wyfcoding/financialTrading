package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// QuoteStrategyRepository 策略仓储接口
type QuoteStrategyRepository interface {
	// SaveStrategy 保存或更新策略
	SaveStrategy(ctx context.Context, strategy *QuoteStrategy) error
	// GetStrategyBySymbol 根据交易对获取策略
	GetStrategyBySymbol(ctx context.Context, symbol string) (*QuoteStrategy, error)
}

// PerformanceRepository 绩效仓储接口
type PerformanceRepository interface {
	// SavePerformance 保存或更新绩效数据
	SavePerformance(ctx context.Context, performance *MarketMakingPerformance) error
	// GetPerformanceBySymbol 根据交易对获取绩效数据
	GetPerformanceBySymbol(ctx context.Context, symbol string) (*MarketMakingPerformance, error)
}

// OrderClient 订单服务客户端接口
type OrderClient interface {
	PlaceOrder(ctx context.Context, symbol string, side string, price, quantity decimal.Decimal) (string, error)
	GetPosition(ctx context.Context, symbol string) (decimal.Decimal, error)
}

// MarketDataClient 市场数据客户端接口
type MarketDataClient interface {
	GetPrice(ctx context.Context, symbol string) (decimal.Decimal, error)
}
