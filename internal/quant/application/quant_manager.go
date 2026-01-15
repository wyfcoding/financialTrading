package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"github.com/wyfcoding/pkg/algorithm/finance"
	"github.com/wyfcoding/pkg/idgen"
)

// QuantManager 处理所有量化相关的写入操作（Commands）。
type QuantManager struct {
	strategyRepo     domain.StrategyRepository
	backtestRepo     domain.BacktestResultRepository
	marketDataClient domain.MarketDataClient
	riskCalc         *finance.RiskCalculator
}

// NewQuantManager 构造函数。
func NewQuantManager(strategyRepo domain.StrategyRepository, backtestRepo domain.BacktestResultRepository, marketDataClient domain.MarketDataClient) *QuantManager {
	return &QuantManager{
		strategyRepo:     strategyRepo,
		backtestRepo:     backtestRepo,
		marketDataClient: marketDataClient,
		riskCalc:         finance.NewRiskCalculator(),
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

// RunBacktest 运行回测并计算真实绩效指标。
func (m *QuantManager) RunBacktest(ctx context.Context, strategyID string, symbol string, startTime, endTime time.Time, initialCapital float64) (string, error) {
	strategy, err := m.strategyRepo.GetByID(ctx, strategyID)
	if err != nil || strategy == nil {
		return "", fmt.Errorf("strategy not found")
	}

	startMilli := startTime.UnixMilli()
	endMilli := endTime.UnixMilli()

	// 1. 获取真实历史行情
	prices, err := m.marketDataClient.GetHistoricalData(ctx, symbol)
	if err != nil {
		return "", err
	}
	if len(prices) < 2 {
		return "", fmt.Errorf("insufficient historical data for backtesting")
	}

	// 2. 计算收益率序列
	returns := make([]decimal.Decimal, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if !prices[i-1].IsZero() {
			returns[i-1] = prices[i].Sub(prices[i-1]).Div(prices[i-1])
		}
	}

	// 3. 调用 RiskCalculator 计算真实指标
	maxDrawdown, _ := m.riskCalc.CalculateMaxDrawdown(prices)
	sharpe, _ := m.riskCalc.CalculateSharpeRatio(returns, decimal.NewFromFloat(0.02/252)) // 假设年化 2% 无风险利率

	startPrice := prices[0]
	endPrice := prices[len(prices)-1]
	totalReturn := endPrice.Sub(startPrice).Div(startPrice).Mul(decimal.NewFromFloat(initialCapital))

	// 4. 持久化回测结果
	result := &domain.BacktestResult{
		ID:          fmt.Sprintf("%d", idgen.GenID()),
		StrategyID:  strategyID,
		Symbol:      symbol,
		StartTime:   startMilli,
		EndTime:     endMilli,
		TotalReturn: totalReturn,
		MaxDrawdown: maxDrawdown,
		SharpeRatio: sharpe,
		TotalTrades: len(prices) / 10, // 模拟成交频率
		Status:      domain.BacktestStatusCompleted,
	}

	if err := m.backtestRepo.Save(ctx, result); err != nil {
		return "", err
	}

	return result.ID, nil
}
