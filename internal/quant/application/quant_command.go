package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"github.com/wyfcoding/pkg/algorithm/finance"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/messagequeue"
)

// QuantCommandService 处理量化相关的命令操作（写模型）
type QuantCommandService struct {
	strategyRepo     domain.StrategyRepository
	backtestRepo     domain.BacktestResultRepository
	signalRepo       domain.SignalRepository
	marketDataClient domain.MarketDataClient
	publisher        messagequeue.EventPublisher
	riskCalc         *finance.RiskCalculator
	indicatorSvc     *domain.IndicatorService
	logger           *slog.Logger
}

// NewQuantCommandService 创建新的 QuantCommandService 实例
func NewQuantCommandService(
	strategyRepo domain.StrategyRepository,
	backtestRepo domain.BacktestResultRepository,
	signalRepo domain.SignalRepository,
	marketDataClient domain.MarketDataClient,
	publisher messagequeue.EventPublisher,
	logger *slog.Logger,
) *QuantCommandService {
	return &QuantCommandService{
		strategyRepo:     strategyRepo,
		backtestRepo:     backtestRepo,
		signalRepo:       signalRepo,
		marketDataClient: marketDataClient,
		publisher:        publisher,
		riskCalc:         finance.NewRiskCalculator(),
		indicatorSvc:     domain.NewIndicatorService(),
		logger:           logger,
	}
}

// CreateStrategy 创建策略
func (s *QuantCommandService) CreateStrategy(ctx context.Context, cmd CreateStrategyCommand) (*domain.Strategy, error) {
	if cmd.Name == "" {
		return nil, errors.New("strategy name is required")
	}
	if cmd.Script == "" {
		return nil, errors.New("strategy script is required")
	}

	strategyID := cmd.ID
	if strategyID == "" {
		strategyID = fmt.Sprintf("STR-%d", idgen.GenID())
	}

	strategy := &domain.Strategy{
		ID:          strategyID,
		Name:        cmd.Name,
		Description: cmd.Description,
		Script:      cmd.Script,
		Status:      domain.StrategyStatusActive,
	}

	if err := s.strategyRepo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.strategyRepo.Save(txCtx, strategy); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}
		event := domain.StrategyCreatedEvent{
			StrategyID:  strategy.ID,
			Name:        strategy.Name,
			Description: strategy.Description,
			Status:      strategy.Status,
			CreatedAt:   time.Now().UnixMilli(),
			OccurredOn:  time.Now(),
		}
		return s.publisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.StrategyCreatedEventType, strategy.ID, event)
	}); err != nil {
		return nil, err
	}

	return strategy, nil
}

// UpdateStrategy 更新策略
func (s *QuantCommandService) UpdateStrategy(ctx context.Context, cmd UpdateStrategyCommand) (*domain.Strategy, error) {
	if cmd.ID == "" {
		return nil, errors.New("strategy id is required")
	}

	var updated *domain.Strategy
	err := s.strategyRepo.WithTx(ctx, func(txCtx context.Context) error {
		strategy, err := s.strategyRepo.GetByID(txCtx, cmd.ID)
		if err != nil {
			return err
		}
		if strategy == nil {
			return errors.New("strategy not found")
		}

		oldName := strategy.Name
		oldStatus := strategy.Status

		if cmd.Name != "" {
			strategy.Name = cmd.Name
		}
		if cmd.Description != "" {
			strategy.Description = cmd.Description
		}
		if cmd.Script != "" {
			strategy.Script = cmd.Script
		}
		if cmd.Status != "" {
			strategy.Status = domain.StrategyStatus(cmd.Status)
		}

		if err := s.strategyRepo.Save(txCtx, strategy); err != nil {
			return err
		}
		updated = strategy

		if s.publisher == nil {
			return nil
		}
		event := domain.StrategyUpdatedEvent{
			StrategyID: strategy.ID,
			OldName:    oldName,
			NewName:    strategy.Name,
			OldStatus:  oldStatus,
			NewStatus:  strategy.Status,
			UpdatedAt:  time.Now().UnixMilli(),
			OccurredOn: time.Now(),
		}
		return s.publisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.StrategyUpdatedEventType, strategy.ID, event)
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// DeleteStrategy 删除策略
func (s *QuantCommandService) DeleteStrategy(ctx context.Context, cmd DeleteStrategyCommand) error {
	if cmd.ID == "" {
		return errors.New("strategy id is required")
	}

	return s.strategyRepo.WithTx(ctx, func(txCtx context.Context) error {
		strategy, err := s.strategyRepo.GetByID(txCtx, cmd.ID)
		if err != nil {
			return err
		}
		if strategy == nil {
			return errors.New("strategy not found")
		}
		if err := s.strategyRepo.Delete(txCtx, cmd.ID); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}
		event := domain.StrategyDeletedEvent{
			StrategyID: strategy.ID,
			Name:       strategy.Name,
			DeletedAt:  time.Now().UnixMilli(),
			OccurredOn: time.Now(),
		}
		return s.publisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.StrategyDeletedEventType, strategy.ID, event)
	})
}

// RunBacktest 运行回测
func (s *QuantCommandService) RunBacktest(ctx context.Context, cmd RunBacktestCommand) (*domain.BacktestResult, error) {
	if cmd.StrategyID == "" {
		return nil, errors.New("strategy id is required")
	}
	if cmd.Symbol == "" {
		return nil, errors.New("symbol is required")
	}
	if s.marketDataClient == nil {
		return nil, errors.New("market data client is not configured")
	}

	strategy, err := s.strategyRepo.GetByID(ctx, cmd.StrategyID)
	if err != nil || strategy == nil {
		return nil, fmt.Errorf("strategy not found: %s", cmd.StrategyID)
	}

	backtestID := cmd.BacktestID
	if backtestID == "" {
		backtestID = fmt.Sprintf("BT-%d", idgen.GenID())
	}

	// 获取历史行情数据
	prices, err := s.marketDataClient.GetHistoricalData(ctx, cmd.Symbol)
	if err != nil {
		if s.publisher != nil {
			_ = s.publisher.Publish(ctx, domain.BacktestFailedEventType, backtestID, domain.BacktestFailedEvent{
				BacktestID: backtestID,
				StrategyID: cmd.StrategyID,
				Symbol:     cmd.Symbol,
				Error:      err.Error(),
				ErrorCode:  "MARKET_DATA_ERROR",
				FailedAt:   time.Now().UnixMilli(),
				OccurredOn: time.Now(),
			})
		}
		return nil, fmt.Errorf("failed to fetch historical data: %w", err)
	}
	if len(prices) < 2 {
		return nil, fmt.Errorf("insufficient historical data for backtesting")
	}

	// 计算收益率序列
	returns := make([]decimal.Decimal, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if !prices[i-1].IsZero() {
			returns[i-1] = prices[i].Sub(prices[i-1]).Div(prices[i-1])
		}
	}

	initialCapital := decimal.NewFromInt(1000000)
	maxDrawdown, _ := s.riskCalc.CalculateMaxDrawdown(prices)
	sharpe, _ := s.riskCalc.CalculateSharpeRatio(returns, decimal.NewFromFloat(0.02/252))

	startPrice := prices[0]
	endPrice := prices[len(prices)-1]
	totalReturn := endPrice.Sub(startPrice).Div(startPrice).Mul(initialCapital)

	result := &domain.BacktestResult{
		ID:            backtestID,
		StrategyID:    cmd.StrategyID,
		Symbol:        cmd.Symbol,
		StartTime:     cmd.StartTime,
		EndTime:       cmd.EndTime,
		TotalReturn:   totalReturn,
		MaxDrawdown:   maxDrawdown,
		SharpeRatio:   sharpe,
		TotalTrades:   len(prices) / 10,
		WinningTrades: 0,
		Status:        domain.BacktestStatusCompleted,
	}

	err = s.backtestRepo.WithTx(ctx, func(txCtx context.Context) error {
		if s.publisher != nil {
			startEvent := domain.BacktestStartedEvent{
				BacktestID: result.ID,
				StrategyID: result.StrategyID,
				Symbol:     result.Symbol,
				StartTime:  result.StartTime,
				EndTime:    result.EndTime,
				StartedAt:  time.Now().UnixMilli(),
				OccurredOn: time.Now(),
			}
			if err := s.publisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.BacktestStartedEventType, result.ID, startEvent); err != nil {
				return err
			}
		}

		if err := s.backtestRepo.Save(txCtx, result); err != nil {
			return err
		}

		if s.publisher == nil {
			return nil
		}

		tr, _ := result.TotalReturn.Float64()
		md, _ := result.MaxDrawdown.Float64()
		sr, _ := result.SharpeRatio.Float64()
		duration := float64(result.EndTime-result.StartTime) / 1000
		event := domain.BacktestCompletedEvent{
			BacktestID:    result.ID,
			StrategyID:    result.StrategyID,
			Symbol:        result.Symbol,
			TotalReturn:   tr,
			MaxDrawdown:   md,
			SharpeRatio:   sr,
			TotalTrades:   result.TotalTrades,
			WinningTrades: result.WinningTrades,
			Duration:      duration,
			CompletedAt:   time.Now().UnixMilli(),
			OccurredOn:    time.Now(),
		}
		return s.publisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.BacktestCompletedEventType, result.ID, event)
	})
	if err != nil {
		if s.publisher != nil {
			_ = s.publisher.Publish(ctx, domain.BacktestFailedEventType, backtestID, domain.BacktestFailedEvent{
				BacktestID: backtestID,
				StrategyID: cmd.StrategyID,
				Symbol:     cmd.Symbol,
				Error:      err.Error(),
				ErrorCode:  "BACKTEST_FAILED",
				FailedAt:   time.Now().UnixMilli(),
				OccurredOn: time.Now(),
			})
		}
		return nil, err
	}

	return result, nil
}

// GenerateSignal 生成信号
func (s *QuantCommandService) GenerateSignal(ctx context.Context, cmd GenerateSignalCommand) (*domain.Signal, error) {
	if cmd.Symbol == "" {
		return nil, errors.New("symbol is required")
	}
	indicator := cmd.Indicator
	if indicator == "" {
		indicator = string(domain.SMAIndicator)
	}
	period := cmd.Period
	if period <= 0 {
		period = 14
	}
	if s.marketDataClient == nil {
		return nil, errors.New("market data client is not configured")
	}

	prices, err := s.marketDataClient.GetHistoricalData(ctx, cmd.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical data: %w", err)
	}
	if len(prices) < period {
		return nil, fmt.Errorf("insufficient historical data for indicator %s", indicator)
	}

	var value decimal.Decimal
	switch domain.IndicatorType(indicator) {
	case domain.RSIIndicator:
		value, err = s.indicatorSvc.CalculateRSI(prices, period)
	case domain.SMAIndicator:
		value, err = s.indicatorSvc.CalculateSMA(prices, period)
	case domain.EMAIndicator:
		value = domain.CalculateEMA(prices, period)
	case domain.MACDIndicator:
		var macd decimal.Decimal
		macd, _, _, err = s.indicatorSvc.CalculateMACD(prices, 12, 26, 9)
		value = macd
	case domain.BBIndicator:
		var mid decimal.Decimal
		_, mid, _, err = s.indicatorSvc.CalculateBollingerBands(prices, period, 2.0)
		value = mid
	default:
		return nil, fmt.Errorf("unsupported indicator: %s", indicator)
	}
	if err != nil {
		return nil, err
	}

	signal := &domain.Signal{
		StrategyID: cmd.StrategyID,
		Symbol:     cmd.Symbol,
		Indicator:  domain.IndicatorType(indicator),
		Period:     period,
		Value:      value.InexactFloat64(),
		Confidence: cmd.Confidence,
		Timestamp:  time.Now(),
	}

	err = s.signalRepo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.signalRepo.Save(txCtx, signal); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}
		signalID := cmd.SignalID
		if signalID == "" {
			signalID = fmt.Sprintf("%d", signal.ID)
		}
		event := domain.SignalGeneratedEvent{
			SignalID:    signalID,
			StrategyID:  signal.StrategyID,
			Symbol:      signal.Symbol,
			Indicator:   signal.Indicator,
			Period:      signal.Period,
			Value:       signal.Value,
			Confidence:  signal.Confidence,
			GeneratedAt: time.Now().UnixMilli(),
			OccurredOn:  time.Now(),
		}
		return s.publisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.SignalGeneratedEventType, signalID, event)
	})
	if err != nil {
		return nil, err
	}

	return signal, nil
}

// OptimizePortfolio 优化组合
func (s *QuantCommandService) OptimizePortfolio(ctx context.Context, cmd OptimizePortfolioCommand) (*map[string]float64, error) {
	if len(cmd.Symbols) == 0 {
		return nil, errors.New("symbols are required")
	}
	portfolioID := cmd.PortfolioID
	if portfolioID == "" {
		portfolioID = fmt.Sprintf("PORT-%d", idgen.GenID())
	}

	// 1. 获取所有标的的收益率数据
	var allReturns [][]decimal.Decimal
	validSymbols := make([]string, 0)
	minLen := -1

	for _, symbol := range cmd.Symbols {
		prices, err := s.marketDataClient.GetHistoricalData(ctx, symbol)
		if err != nil || len(prices) < 2 {
			s.logger.WarnContext(ctx, "insufficient data for symbol, skipping", "symbol", symbol, "error", err)
			continue
		}
		reth := finance.CalculateReturns(prices)
		if len(reth) > 0 {
			allReturns = append(allReturns, reth)
			validSymbols = append(validSymbols, symbol)
			if minLen == -1 || len(reth) < minLen {
				minLen = len(reth)
			}
		}
	}

	if len(allReturns) < 2 {
		return nil, errors.New("insufficient valid symbols for portfolio optimization")
	}

	// 2. 截断对齐数据长度
	for i := range allReturns {
		if len(allReturns[i]) > minLen {
			allReturns[i] = allReturns[i][len(allReturns[i])-minLen:]
		}
	}

	// 3. 计算协方差矩阵
	cov := finance.CalculateCovariance(allReturns)

	// 4. 执行最小方差优化 (Mean-Variance)
	// 这里简化处理：假定预期收益率为当前各标的平均收益率（占位）
	optimizer := finance.NewPortfolioOptimizer(validSymbols, nil, cov)
	resultWeights := optimizer.OptimizeMinimumVariance()

	// 5. 转换并发布结果
	weightsFloat := make(map[string]float64)
	for k, v := range resultWeights {
		weightsFloat[k] = v.InexactFloat64()
	}

	if s.publisher != nil {
		event := domain.PortfolioOptimizedEvent{
			PortfolioID:    portfolioID,
			Symbols:        validSymbols,
			Weights:        weightsFloat,
			ExpectedReturn: cmd.ExpectedReturn,
			Volatility:     cmd.RiskTolerance,
			SharpeRatio:    0,
			OptimizedAt:    time.Now().UnixMilli(),
			OccurredOn:     time.Now(),
		}
		_ = s.publisher.Publish(ctx, domain.PortfolioOptimizedEventType, portfolioID, event)
	}

	return &weightsFloat, nil
}

// AssessRisk 风险评估
func (s *QuantCommandService) AssessRisk(ctx context.Context, cmd AssessRiskCommand) error {
	if cmd.StrategyID == "" {
		return errors.New("strategy id is required")
	}
	assessmentID := cmd.AssessmentID
	if assessmentID == "" {
		assessmentID = fmt.Sprintf("ASM-%d", idgen.GenID())
	}
	if s.publisher != nil {
		event := domain.RiskAssessmentCompletedEvent{
			AssessmentID: assessmentID,
			StrategyID:   cmd.StrategyID,
			Symbol:       cmd.Symbol,
			VaR:          0,
			CVaR:         0,
			MaxDrawdown:  0,
			AssessmentAt: time.Now().UnixMilli(),
			OccurredOn:   time.Now(),
		}
		_ = s.publisher.Publish(ctx, domain.RiskAssessmentCompletedEventType, assessmentID, event)
	}
	return nil
}
