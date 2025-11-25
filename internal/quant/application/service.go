package application

import (
	"context"
	"fmt"
	"time"

	"github.com/fynnwu/FinancialTrading/internal/quant/domain"
	"github.com/fynnwu/FinancialTrading/pkg/logger"
	"github.com/google/uuid"
)

// QuantService 应用服务
type QuantService struct {
	strategyRepo     domain.StrategyRepository
	backtestRepo     domain.BacktestResultRepository
	marketDataClient domain.MarketDataClient
}

// NewQuantService 创建应用服务实例
func NewQuantService(strategyRepo domain.StrategyRepository, backtestRepo domain.BacktestResultRepository, marketDataClient domain.MarketDataClient) *QuantService {
	return &QuantService{
		strategyRepo:     strategyRepo,
		backtestRepo:     backtestRepo,
		marketDataClient: marketDataClient,
	}
}

// CreateStrategy 创建策略
func (s *QuantService) CreateStrategy(ctx context.Context, name string, description string, script string) (string, error) {
	strategy := &domain.Strategy{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Script:      script,
		Status:      domain.StrategyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.strategyRepo.Save(ctx, strategy); err != nil {
		logger.Error(ctx, "Failed to save strategy",
			"name", name,
			"error", err,
		)
		return "", fmt.Errorf("failed to save strategy: %w", err)
	}

	logger.Info(ctx, "Strategy created",
		"strategy_id", strategy.ID,
		"name", name,
	)

	return strategy.ID, nil
}

// GetStrategy 获取策略
func (s *QuantService) GetStrategy(ctx context.Context, id string) (*domain.Strategy, error) {
	strategy, err := s.strategyRepo.GetByID(ctx, id)
	if err != nil {
		logger.Error(ctx, "Failed to get strategy",
			"strategy_id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get strategy: %w", err)
	}
	return strategy, nil
}

// RunBacktest 运行回测
func (s *QuantService) RunBacktest(ctx context.Context, strategyID string, symbol string, startTime, endTime time.Time, initialCapital float64) (string, error) {
	// 1. 获取策略
	strategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		logger.Error(ctx, "Failed to get strategy for backtest",
			"strategy_id", strategyID,
			"error", err,
		)
		return "", fmt.Errorf("failed to get strategy: %w", err)
	}
	if strategy == nil {
		return "", fmt.Errorf("strategy not found: %s", strategyID)
	}

	// 2. 获取历史数据
	prices, err := s.marketDataClient.GetHistoricalData(ctx, symbol, startTime, endTime)
	if err != nil {
		logger.Error(ctx, "Failed to get historical data",
			"symbol", symbol,
			"start_time", startTime,
			"end_time", endTime,
			"error", err,
		)
		return "", fmt.Errorf("failed to get historical data: %w", err)
	}

	// 3. 执行回测逻辑（简化版）
	// 假设策略是：如果价格上涨就买入，下跌就卖出
	// 这是一个非常简单的模拟
	totalReturn := 0.0
	if len(prices) > 1 {
		startPrice := prices[0]
		endPrice := prices[len(prices)-1]
		totalReturn = (endPrice - startPrice) / startPrice * initialCapital
	}

	// 4. 保存回测结果
	result := &domain.BacktestResult{
		ID:          uuid.New().String(),
		StrategyID:  strategyID,
		Symbol:      symbol,
		StartTime:   startTime,
		EndTime:     endTime,
		TotalReturn: totalReturn,
		MaxDrawdown: 0.1, // 模拟值
		SharpeRatio: 1.5, // 模拟值
		TotalTrades: 10,  // 模拟值
		Status:      domain.BacktestStatusCompleted,
		CreatedAt:   time.Now(),
	}

	if err := s.backtestRepo.Save(ctx, result); err != nil {
		logger.Error(ctx, "Failed to save backtest result",
			"strategy_id", strategyID,
			"symbol", symbol,
			"error", err,
		)
		return "", fmt.Errorf("failed to save backtest result: %w", err)
	}

	logger.Info(ctx, "Backtest completed",
		"backtest_id", result.ID,
		"strategy_id", strategyID,
		"symbol", symbol,
		"total_return", totalReturn,
	)

	return result.ID, nil
}

// GetBacktestResult 获取回测结果
func (s *QuantService) GetBacktestResult(ctx context.Context, id string) (*domain.BacktestResult, error) {
	result, err := s.backtestRepo.GetByID(ctx, id)
	if err != nil {
		logger.Error(ctx, "Failed to get backtest result",
			"result_id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get backtest result: %w", err)
	}
	return result, nil
}
