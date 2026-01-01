package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

// MarketMakingManager 处理所有做市相关的写入操作（Commands）。
type MarketMakingManager struct {
	strategyRepo     domain.QuoteStrategyRepository
	performanceRepo  domain.PerformanceRepository
	orderClient      domain.OrderClient
	marketDataClient domain.MarketDataClient
}

// NewMarketMakingManager 构造函数。
func NewMarketMakingManager(
	strategyRepo domain.QuoteStrategyRepository,
	performanceRepo domain.PerformanceRepository,
	orderClient domain.OrderClient,
	marketDataClient domain.MarketDataClient,
) *MarketMakingManager {
	return &MarketMakingManager{
		strategyRepo:     strategyRepo,
		performanceRepo:  performanceRepo,
		orderClient:      orderClient,
		marketDataClient: marketDataClient,
	}
}

// SetStrategy 设置做市策略
func (m *MarketMakingManager) SetStrategy(ctx context.Context, symbol string, spread, minOrderSize, maxOrderSize, maxPosition decimal.Decimal, status string) (string, error) {
	strategy := &domain.QuoteStrategy{
		Symbol:       symbol,
		Spread:       spread,
		MinOrderSize: minOrderSize,
		MaxOrderSize: maxOrderSize,
		MaxPosition:  maxPosition,
		Status:       domain.StrategyStatus(status),
	}

	if err := m.strategyRepo.SaveStrategy(ctx, strategy); err != nil {
		return "", err
	}

	// 如果策略激活，启动做市逻辑（简化模拟）
	if strategy.Status == domain.StrategyStatusActive {
		go m.runMarketMaking(symbol)
	}

	return strategy.Symbol, nil
}

// runMarketMaking 运行做市逻辑（简化模拟）
func (m *MarketMakingManager) runMarketMaking(symbol string) {
	ctx := context.Background()
	price, err := m.marketDataClient.GetPrice(ctx, symbol)
	if err != nil {
		return
	}

	strategy, err := m.strategyRepo.GetStrategyBySymbol(ctx, symbol)
	if err != nil || strategy == nil || strategy.Status != domain.StrategyStatusActive {
		return
	}

	halfSpread := strategy.Spread.Div(decimal.NewFromInt(2))
	bidPrice := price.Mul(decimal.NewFromInt(1).Sub(halfSpread))
	askPrice := price.Mul(decimal.NewFromInt(1).Add(halfSpread))
	quantity := strategy.MinOrderSize

	m.orderClient.PlaceOrder(ctx, symbol, "BUY", bidPrice, quantity)
	m.orderClient.PlaceOrder(ctx, symbol, "SELL", askPrice, quantity)

	perf := &domain.MarketMakingPerformance{
		Symbol:      symbol,
		TotalPnL:    decimal.NewFromFloat(100.0),
		TotalVolume: quantity.Mul(decimal.NewFromInt(2)),
		TotalTrades: 2,
		SharpeRatio: decimal.NewFromFloat(2.0),
		StartTime:   time.Now(),
		EndTime:     time.Now(),
	}
	m.performanceRepo.SavePerformance(ctx, perf)
}
