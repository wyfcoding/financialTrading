// 包 做市服务的用例逻辑
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
	"github.com/wyfcoding/pkg/logging"
)

// MarketMakingService 做市应用服务
// 负责管理做市策略、执行做市逻辑以及监控做市绩效
type MarketMakingService struct {
	strategyRepo     domain.QuoteStrategyRepository // 策略仓储接口
	performanceRepo  domain.PerformanceRepository   // 绩效仓储接口
	orderClient      domain.OrderClient             // 订单服务客户端
	marketDataClient domain.MarketDataClient        // 市场数据服务客户端
}

// NewMarketMakingService 创建做市应用服务实例
// strategyRepo: 注入的策略仓储实现
// performanceRepo: 注入的绩效仓储实现
// orderClient: 注入的订单服务客户端
// marketDataClient: 注入的市场数据服务客户端
func NewMarketMakingService(
	strategyRepo domain.QuoteStrategyRepository,
	performanceRepo domain.PerformanceRepository,
	orderClient domain.OrderClient,
	marketDataClient domain.MarketDataClient,
) *MarketMakingService {
	return &MarketMakingService{
		strategyRepo:     strategyRepo,
		performanceRepo:  performanceRepo,
		orderClient:      orderClient,
		marketDataClient: marketDataClient,
	}
}

// SetStrategy 设置做市策略
func (s *MarketMakingService) SetStrategy(ctx context.Context, symbol string, spread, minOrderSize, maxOrderSize, maxPosition decimal.Decimal, status string) (string, error) {
	strategy := &domain.QuoteStrategy{
		Symbol:       symbol,
		Spread:       spread,
		MinOrderSize: minOrderSize,
		MaxOrderSize: maxOrderSize,
		MaxPosition:  maxPosition,
		Status:       domain.StrategyStatus(status),
	}

	if err := s.strategyRepo.SaveStrategy(ctx, strategy); err != nil {
		logging.Error(ctx, "Failed to save strategy",
			"symbol", symbol,
			"error", err,
		)
		return "", fmt.Errorf("failed to save strategy: %w", err)
	}

	logging.Info(ctx, "Strategy set",
		"symbol", symbol,
		"status", status,
	)

	// 如果策略激活，启动做市逻辑（简化版：仅启动一个goroutine模拟）
	if strategy.Status == domain.StrategyStatusActive {
		go s.runMarketMaking(symbol)
	}

	return strategy.Symbol, nil
}

// GetStrategy 获取做市策略
func (s *MarketMakingService) GetStrategy(ctx context.Context, symbol string) (*domain.QuoteStrategy, error) {
	strategy, err := s.strategyRepo.GetStrategyBySymbol(ctx, symbol)
	if err != nil {
		logging.Error(ctx, "Failed to get strategy",
			"symbol", symbol,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get strategy: %w", err)
	}
	return strategy, nil
}

// GetPerformance 获取做市绩效
func (s *MarketMakingService) GetPerformance(ctx context.Context, symbol string) (*domain.MarketMakingPerformance, error) {
	performance, err := s.performanceRepo.GetPerformanceBySymbol(ctx, symbol)
	if err != nil {
		logging.Error(ctx, "Failed to get performance",
			"symbol", symbol,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get performance: %w", err)
	}
	return performance, nil
}

// runMarketMaking 运行做市逻辑（简化模拟）
func (s *MarketMakingService) runMarketMaking(symbol string) {
	// 实际逻辑会复杂得多，这里仅模拟下单和更新绩效
	ctx := context.Background()

	// 1. 获取当前价格
	price, _ := s.marketDataClient.GetPrice(ctx, symbol)

	// 2. 获取策略
	strategy, _ := s.strategyRepo.GetStrategyBySymbol(ctx, symbol)
	if strategy == nil || strategy.Status != domain.StrategyStatusActive {
		return
	}

	// 3. 计算买卖单价格
	halfSpread := strategy.Spread.Div(decimal.NewFromInt(2))
	bidPrice := price.Mul(decimal.NewFromInt(1).Sub(halfSpread))
	askPrice := price.Mul(decimal.NewFromInt(1).Add(halfSpread))
	quantity := strategy.MinOrderSize

	// 4. 下单
	if _, err := s.orderClient.PlaceOrder(ctx, symbol, "BUY", bidPrice, quantity); err != nil {
		logging.Error(ctx, "Failed to place buy order",
			"symbol", symbol,
			"price", bidPrice,
			"quantity", quantity,
			"error", err,
		)
	}
	if _, err := s.orderClient.PlaceOrder(ctx, symbol, "SELL", askPrice, quantity); err != nil {
		logging.Error(ctx, "Failed to place sell order",
			"symbol", symbol,
			"price", askPrice,
			"quantity", quantity,
			"error", err,
		)
	}

	// 5. 更新绩效（模拟）
	perf := &domain.MarketMakingPerformance{
		Symbol:      symbol,
		TotalPnL:    decimal.NewFromFloat(100.0), // 模拟盈利
		TotalVolume: quantity.Mul(decimal.NewFromInt(2)),
		TotalTrades: 2,
		SharpeRatio: decimal.NewFromFloat(2.0),
		StartTime:   time.Now(),
		EndTime:     time.Now(),
	}
	if err := s.performanceRepo.SavePerformance(ctx, perf); err != nil {
		logging.Error(ctx, "Failed to save performance",
			"symbol", symbol,
			"error", err,
		)
	}
}
