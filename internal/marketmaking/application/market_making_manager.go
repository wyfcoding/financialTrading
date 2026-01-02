package application

import (
	"context"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
	"github.com/wyfcoding/pkg/logging"
)

// MarketMakingManager 处理所有做市相关的写入操作（Commands）。
type MarketMakingManager struct {
	strategyRepo     domain.QuoteStrategyRepository
	performanceRepo  domain.PerformanceRepository
	orderClient      domain.OrderClient
	marketDataClient domain.MarketDataClient
	
	activeTasks map[string]context.CancelFunc // 追踪活跃的做市任务
	mu          sync.RWMutex
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
		activeTasks:      make(map[string]context.CancelFunc),
	}
}

// SetStrategy 设置做市策略并根据状态启动/停止任务
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

	// 动态管理任务生命周期
	if strategy.Status == domain.StrategyStatusActive {
		m.startMarketMaking(symbol)
	} else {
		m.stopMarketMaking(symbol)
	}

	return strategy.Symbol, nil
}

func (m *MarketMakingManager) startMarketMaking(symbol string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.activeTasks[symbol]; exists {
		return // 任务已在运行
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.activeTasks[symbol] = cancel

	go m.runMarketMakingLoop(ctx, symbol)
	logging.Info(context.Background(), "MarketMaking: started continuous loop", "symbol", symbol)
}

func (m *MarketMakingManager) stopMarketMaking(symbol string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cancel, ok := m.activeTasks[symbol]; ok {
		cancel()
		delete(m.activeTasks, symbol)
		logging.Info(context.Background(), "MarketMaking: stopped loop", "symbol", symbol)
	}
}

// runMarketMakingLoop 持续运行做市逻辑
func (m *MarketMakingManager) runMarketMakingLoop(ctx context.Context, symbol string) {
	// 每 5 秒尝试更新一次报价
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.executeQuote(ctx, symbol)
		}
	}
}

func (m *MarketMakingManager) executeQuote(ctx context.Context, symbol string) {
	// 1. 获取最新市场价
	price, err := m.marketDataClient.GetPrice(ctx, symbol)
	if err != nil {
		return
	}

	// 2. 获取当前策略配置
	strategy, err := m.strategyRepo.GetStrategyBySymbol(ctx, symbol)
	if err != nil || strategy == nil || strategy.Status != domain.StrategyStatusActive {
		m.stopMarketMaking(symbol) // 如果策略失效，自动停止循环
		return
	}

	// 3. 计算双向报价
	halfSpread := strategy.Spread.Div(decimal.NewFromInt(2))
	bidPrice := price.Mul(decimal.NewFromInt(1).Sub(halfSpread))
	askPrice := price.Mul(decimal.NewFromInt(1).Add(halfSpread))
	quantity := strategy.MinOrderSize

	// 4. 执行下单
	_, err1 := m.orderClient.PlaceOrder(ctx, symbol, "BUY", bidPrice, quantity)
	_, err2 := m.orderClient.PlaceOrder(ctx, symbol, "SELL", askPrice, quantity)

	// 5. 记录绩效
	if err1 == nil && err2 == nil {
		perf := &domain.MarketMakingPerformance{
			Symbol:      symbol,
			TotalVolume: quantity.Mul(decimal.NewFromInt(2)),
			TotalTrades: 2,
			EndTime:     time.Now(),
		}
		_ = m.performanceRepo.SavePerformance(ctx, perf)
	}
}
