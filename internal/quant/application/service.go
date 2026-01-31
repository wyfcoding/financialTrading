package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/quant/domain"
)

// mockEventPublisher 事件发布者的空实现
type mockEventPublisher struct{}

func (m *mockEventPublisher) PublishStrategyCreated(event domain.StrategyCreatedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishStrategyUpdated(event domain.StrategyUpdatedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishStrategyDeleted(event domain.StrategyDeletedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishBacktestCompleted(event domain.BacktestCompletedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishSignalGenerated(event domain.SignalGeneratedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishPortfolioOptimized(event domain.PortfolioOptimizedEvent) error {
	return nil
}

// QuantService 量化服务门面，整合命令和查询服务
type QuantService struct {
	Command *QuantCommand
	Query   *QuantQuery
}

// NewQuantService 构造函数
func NewQuantService(
	strategyRepo domain.StrategyRepository,
	backtestRepo domain.BacktestResultRepository,
	signalRepo domain.SignalRepository,
	db interface{},
) (*QuantService, error) {
	// 创建命令服务
	command := NewQuantCommand(strategyRepo)

	// 创建查询服务
	query := NewQuantQuery(strategyRepo, backtestRepo, signalRepo)

	return &QuantService{
		Command: command,
		Query:   query,
	}, nil
}

// --- Command (Writes) ---

// CreateStrategy 创建策略
func (s *QuantService) CreateStrategy(ctx context.Context, cmd CreateStrategyCommand) (*domain.Strategy, error) {
	return s.Command.CreateStrategy(ctx, cmd)
}

// UpdateStrategy 更新策略
func (s *QuantService) UpdateStrategy(ctx context.Context, cmd UpdateStrategyCommand) (*domain.Strategy, error) {
	return s.Command.UpdateStrategy(ctx, cmd)
}

// DeleteStrategy 删除策略
func (s *QuantService) DeleteStrategy(ctx context.Context, cmd DeleteStrategyCommand) error {
	return s.Command.DeleteStrategy(ctx, cmd)
}

// RunBacktest 运行回测
func (s *QuantService) RunBacktest(ctx context.Context, cmd RunBacktestCommand) (*domain.BacktestResult, error) {
	return s.Command.RunBacktest(ctx, cmd)
}

// GenerateSignal 生成信号
func (s *QuantService) GenerateSignal(ctx context.Context, cmd GenerateSignalCommand) error {
	return s.Command.GenerateSignal(ctx, cmd)
}

// OptimizePortfolio 优化组合
func (s *QuantService) OptimizePortfolio(ctx context.Context, cmd OptimizePortfolioCommand) error {
	return s.Command.OptimizePortfolio(ctx, cmd)
}

// AssessRisk 风险评估
func (s *QuantService) AssessRisk(ctx context.Context, cmd AssessRiskCommand) error {
	return s.Command.AssessRisk(ctx, cmd)
}

// --- Query (Reads) ---

// GetStrategy 获取策略
func (s *QuantService) GetStrategy(ctx context.Context, id string) (*domain.Strategy, error) {
	return s.Query.GetStrategy(ctx, id)
}

// GetBacktestResult 获取回测结果
func (s *QuantService) GetBacktestResult(ctx context.Context, id string) (*domain.BacktestResult, error) {
	return s.Query.GetBacktestResult(ctx, id)
}

// GetSignal 获取信号
func (s *QuantService) GetSignal(ctx context.Context, symbol string, indicator string, period int) (*SignalDTO, error) {
	return s.Query.GetSignal(ctx, symbol, indicator, period)
}
