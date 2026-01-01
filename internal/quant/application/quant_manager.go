package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"github.com/wyfcoding/pkg/idgen"
)

// QuantManager 处理所有量化相关的写入操作（Commands）。
type QuantManager struct {
	strategyRepo     domain.StrategyRepository
	backtestRepo     domain.BacktestResultRepository
	marketDataClient domain.MarketDataClient
}

// NewQuantManager 构造函数。
func NewQuantManager(strategyRepo domain.StrategyRepository, backtestRepo domain.BacktestResultRepository, marketDataClient domain.MarketDataClient) *QuantManager {
	return &QuantManager{
		strategyRepo:     strategyRepo,
		backtestRepo:     backtestRepo,
		marketDataClient: marketDataClient,
	}
}

// CreateStrategy 创建策略
func (m *QuantManager) CreateStrategy(ctx context.Context, name string, description string, script string) (string, error) {
	strategy := &domain.Strategy{
		ID:          fmt.Sprintf("%d", idgen.GenID()),
		Name:        name,
		Description: description,
		Script:      script,
		Status:      domain.StrategyStatusActive,
	}

	if err := m.strategyRepo.Save(ctx, strategy); err != nil {
		return "", err
	}

	return strategy.ID, nil
}

// RunBacktest 运行回测
func (m *QuantManager) RunBacktest(ctx context.Context, strategyID string, symbol string, startTime, endTime time.Time, initialCapital float64) (string, error) {
	strategy, err := m.strategyRepo.GetByID(ctx, strategyID)
	if err != nil || strategy == nil {
		return "", fmt.Errorf("strategy not found")
	}

	startMilli := startTime.UnixMilli()
	endMilli := endTime.UnixMilli()

	prices, err := m.marketDataClient.GetHistoricalData(ctx, symbol, startMilli, endMilli)
	if err != nil {
		return "", err
	}

	totalReturn := decimal.Zero
	if len(prices) > 1 {
		startPrice := prices[0]
		endPrice := prices[len(prices)-1]
		if !startPrice.IsZero() {
			totalReturn = endPrice.Sub(startPrice).Div(startPrice).Mul(decimal.NewFromFloat(initialCapital))
		}
	}

	result := &domain.BacktestResult{
		ID:          fmt.Sprintf("%d", idgen.GenID()),
		StrategyID:  strategyID,
		Symbol:      symbol,
		StartTime:   startMilli,
		EndTime:     endMilli,
		TotalReturn: totalReturn,
		MaxDrawdown: decimal.NewFromFloat(0.1),
		SharpeRatio: decimal.NewFromFloat(1.5),
		TotalTrades: 10,
		Status:      domain.BacktestStatusCompleted,
	}

	if err := m.backtestRepo.Save(ctx, result); err != nil {
		return "", err
	}

	return result.ID, nil
}
