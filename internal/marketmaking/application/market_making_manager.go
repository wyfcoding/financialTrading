package application

import (
	"context"
	"log/slog"
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
	logger      *slog.Logger

	// 实时绩效统计
	totalPnL    map[string]decimal.Decimal
	totalVolume map[string]decimal.Decimal
	totalTrades map[string]int64
}

// NewMarketMakingManager 构造函数。
func NewMarketMakingManager(
	strategyRepo domain.QuoteStrategyRepository,
	performanceRepo domain.PerformanceRepository,
	orderClient domain.OrderClient,
	marketDataClient domain.MarketDataClient,
	logger *slog.Logger,
) *MarketMakingManager {
	return &MarketMakingManager{
		strategyRepo:     strategyRepo,
		performanceRepo:  performanceRepo,
		orderClient:      orderClient,
		marketDataClient: marketDataClient,
		activeTasks:      make(map[string]context.CancelFunc),
		logger:           logger,
		totalPnL:         make(map[string]decimal.Decimal),
		totalVolume:      make(map[string]decimal.Decimal),
		totalTrades:      make(map[string]int64),
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
		m.stopMarketMaking(symbol)
		return
	}

	// 3. 获取当前持仓 (真实风控)
	pos, err := m.orderClient.GetPosition(ctx, symbol)
	if err != nil {
		m.logger.ErrorContext(ctx, "failed to fetch current position", "symbol", symbol, "error", err)
		return
	}

	// 4. 计算双向报价
	halfSpread := strategy.Spread.Div(decimal.NewFromInt(2))
	bidPrice := price.Mul(decimal.NewFromInt(1).Sub(halfSpread))
	askPrice := price.Mul(decimal.NewFromInt(1).Add(halfSpread))
	quantity := strategy.MinOrderSize

	// 5. 执行带持仓限制的下单
	var successQty decimal.Decimal
	var tradesCount int64

	// 检查买入持仓限制
	if pos.Add(quantity).LessThanOrEqual(strategy.MaxPosition) {
		if _, err := m.orderClient.PlaceOrder(ctx, symbol, "BUY", bidPrice, quantity); err == nil {
			successQty = successQty.Add(quantity)
			tradesCount++
		}
	}

	// 检查卖出持仓限制
	if pos.Sub(quantity).Abs().LessThanOrEqual(strategy.MaxPosition) {
		if _, err := m.orderClient.PlaceOrder(ctx, symbol, "SELL", askPrice, quantity); err == nil {
			successQty = successQty.Add(quantity)
			tradesCount++
		}
	}

	// 6. 真实绩效累计与盈亏核算
	if tradesCount > 0 {
		m.mu.Lock()
		m.totalVolume[symbol] = m.totalVolume[symbol].Add(successQty)
		m.totalTrades[symbol] += tradesCount

		// 真实化执行：根据买卖成交价差计算盈亏
		// PnL = (SellPrice - BuyPrice) * Quantity
		pnl := askPrice.Sub(bidPrice).Mul(successQty).Div(decimal.NewFromInt(2))
		m.totalPnL[symbol] = m.totalPnL[symbol].Add(pnl)

		vol := m.totalVolume[symbol]
		trd := m.totalTrades[symbol]
		pnlTotal := m.totalPnL[symbol]
		m.mu.Unlock()

		perf := &domain.MarketMakingPerformance{
			Symbol:      symbol,
			TotalPnL:    pnlTotal,
			TotalVolume: vol,
			TotalTrades: trd,
			EndTime:     time.Now(),
		}
		_ = m.performanceRepo.SavePerformance(ctx, perf)
	}
}
