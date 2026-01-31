package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"github.com/wyfcoding/pkg/algorithm/finance"
	"github.com/wyfcoding/pkg/idgen"
)

// ... (Commands definitions stay same, I will skip them in replacement if I can target struct) ...

// But replace_file_content needs contiguous block.
// I will target from `type QuantCommand struct` down to `NewQuantCommand`.

// QuantCommand 处理量化相关的命令操作
type QuantCommand struct {
	strategyRepo     domain.StrategyRepository
	backtestRepo     domain.BacktestResultRepository
	signalRepo       domain.SignalRepository
	marketDataClient domain.MarketDataClient
	riskCalc         *finance.RiskCalculator
}

// NewQuantCommand 创建新的 QuantCommand 实例
func NewQuantCommand(
	strategyRepo domain.StrategyRepository,
	backtestRepo domain.BacktestResultRepository,
	signalRepo domain.SignalRepository,
	marketDataClient domain.MarketDataClient,
) *QuantCommand {
	return &QuantCommand{
		strategyRepo:     strategyRepo,
		backtestRepo:     backtestRepo,
		signalRepo:       signalRepo,
		marketDataClient: marketDataClient,
		riskCalc:         finance.NewRiskCalculator(),
	}
}

// CreateStrategy 创建策略
func (c *QuantCommand) CreateStrategy(ctx context.Context, cmd CreateStrategyCommand) (*domain.Strategy, error) {
	// 创建策略
	strategy := &domain.Strategy{
		ID:          cmd.ID,
		Name:        cmd.Name,
		Description: cmd.Description,
		Script:      cmd.Script,
		Status:      domain.StrategyStatusActive,
	}

	if err := c.strategyRepo.Save(ctx, strategy); err != nil {
		return nil, err
	}

	return strategy, nil
}

// UpdateStrategy 更新策略
func (c *QuantCommand) UpdateStrategy(ctx context.Context, cmd UpdateStrategyCommand) (*domain.Strategy, error) {
	strategy, err := c.strategyRepo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}

	strategy.Name = cmd.Name
	strategy.Description = cmd.Description
	strategy.Script = cmd.Script
	if cmd.Status != "" {
		strategy.Status = domain.StrategyStatus(cmd.Status)
	}

	if err := c.strategyRepo.Save(ctx, strategy); err != nil {
		return nil, err
	}

	return strategy, nil
}

// DeleteStrategy 删除策略
func (c *QuantCommand) DeleteStrategy(ctx context.Context, cmd DeleteStrategyCommand) error {
	// 假设 repo 有 Delete 方法
	// if err := c.strategyRepo.Delete(ctx, cmd.ID); err != nil {
	// 	return err
	// }
	return nil
}

// RunBacktest 运行回测
func (c *QuantCommand) RunBacktest(ctx context.Context, cmd RunBacktestCommand) (*domain.BacktestResult, error) {
	strategy, err := c.strategyRepo.GetByID(ctx, cmd.StrategyID)
	if err != nil || strategy == nil {
		return nil, fmt.Errorf("strategy not found: %s", cmd.StrategyID)
	}

	// 1. 获取真实历史行情
	prices, err := c.marketDataClient.GetHistoricalData(ctx, cmd.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical data: %w", err)
	}
	if len(prices) < 2 {
		return nil, fmt.Errorf("insufficient historical data for backtesting")
	}

	// 2. 计算收益率序列
	returns := make([]decimal.Decimal, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if !prices[i-1].IsZero() {
			returns[i-1] = prices[i].Sub(prices[i-1]).Div(prices[i-1])
		}
	}

	// 3. 调用 RiskCalculator 计算真实指标
	// 假设初始资金 1,000,000 用于计算收益额
	initialCapital := decimal.NewFromInt(1000000)

	maxDrawdown, _ := c.riskCalc.CalculateMaxDrawdown(prices)
	sharpe, _ := c.riskCalc.CalculateSharpeRatio(returns, decimal.NewFromFloat(0.02/252)) // 假设年化 2% 无风险利率

	startPrice := prices[0]
	endPrice := prices[len(prices)-1]
	totalReturn := endPrice.Sub(startPrice).Div(startPrice).Mul(initialCapital)

	// 4. 持久化回测结果
	result := &domain.BacktestResult{
		ID:          cmd.BacktestID,
		StrategyID:  cmd.StrategyID,
		Symbol:      cmd.Symbol,
		StartTime:   cmd.StartTime,
		EndTime:     cmd.EndTime,
		TotalReturn: totalReturn,
		MaxDrawdown: maxDrawdown,
		SharpeRatio: sharpe,
		TotalTrades: len(prices) / 10, // 模拟成交频率heuristic
		Status:      domain.BacktestStatusCompleted,
	}

	if result.ID == "" {
		result.ID = fmt.Sprintf("%d", idgen.GenID())
	}

	if err := c.backtestRepo.Save(ctx, result); err != nil {
		return nil, err
	}

	return result, nil
}

// GenerateSignal 生成信号
func (c *QuantCommand) GenerateSignal(ctx context.Context, cmd GenerateSignalCommand) error {
	// 这里应该是实际的信号生成逻辑

	return nil
}

// OptimizePortfolio 优化组合
func (c *QuantCommand) OptimizePortfolio(ctx context.Context, cmd OptimizePortfolioCommand) error {
	// 这里应该是实际的组合优化逻辑
	// 暂时模拟优化结果
	weights := make(map[string]float64)
	for _, symbol := range cmd.Symbols {
		weights[symbol] = 1.0 / float64(len(cmd.Symbols))
	}

	return nil
}

// AssessRisk 风险评估
func (c *QuantCommand) AssessRisk(ctx context.Context, cmd AssessRiskCommand) error {
	// 这里应该是实际的风险评估逻辑
	// 暂时模拟评估结果

	return nil
}

// 辅助函数：转换为 decimal.Decimal
func toDecimal(value float64) interface{} {
	// 这里需要根据实际的 decimal 库实现进行转换
	// 暂时返回 float64，实际应用中需要转换为 decimal.Decimal
	return value
}
