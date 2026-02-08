// Package application 算法交易应用层
package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/algotrading/domain"
	"github.com/wyfcoding/pkg/messagequeue"
)

// CommandService 算法交易命令服务
type CommandService struct {
	strategyRepo   domain.StrategyRepository
	backtestRepo   domain.BacktestRepository
	eventPublisher messagequeue.EventPublisher
	logger         *slog.Logger
}

// NewCommandService 创建命令服务
func NewCommandService(
	strategyRepo domain.StrategyRepository,
	backtestRepo domain.BacktestRepository,
	eventPublisher messagequeue.EventPublisher,
	logger *slog.Logger,
) *CommandService {
	return &CommandService{
		strategyRepo:   strategyRepo,
		backtestRepo:   backtestRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// CreateStrategyCommand 创建策略命令
type CreateStrategyCommand struct {
	UserID     uint64
	Type       domain.StrategyType
	Symbol     string
	Parameters string
}

// CreateStrategy 创建策略
func (s *CommandService) CreateStrategy(ctx context.Context, cmd CreateStrategyCommand) (string, error) {
	now := time.Now()
	strategyID := fmt.Sprintf("ST%s%04d", now.Format("20060102150405"), now.UnixNano()%10000)

	strategy := domain.NewStrategy(
		strategyID,
		cmd.UserID,
		cmd.Type,
		cmd.Symbol,
		cmd.Parameters,
	)

	if err := s.strategyRepo.Save(ctx, strategy); err != nil {
		return "", err
	}

	s.logger.InfoContext(ctx, "strategy created", "strategy_id", strategyID)
	return strategyID, nil
}

// StartStrategy 启动策略
func (s *CommandService) StartStrategy(ctx context.Context, strategyID string) error {
	strategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return err
	}

	if err := strategy.Start(); err != nil {
		return err
	}

	if err := s.strategyRepo.Save(ctx, strategy); err != nil {
		return err
	}

	s.publishEvents(ctx, strategy.GetDomainEvents())
	strategy.ClearDomainEvents()

	s.logger.InfoContext(ctx, "strategy started", "strategy_id", strategyID)
	// 实际应异步启动执行引擎或发送命令给引擎
	return nil
}

// StopStrategy 停止策略
func (s *CommandService) StopStrategy(ctx context.Context, strategyID string) error {
	strategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return err
	}

	if err := strategy.Stop(); err != nil {
		return err
	}

	if err := s.strategyRepo.Save(ctx, strategy); err != nil {
		return err
	}

	s.publishEvents(ctx, strategy.GetDomainEvents())
	strategy.ClearDomainEvents()

	s.logger.InfoContext(ctx, "strategy stopped", "strategy_id", strategyID)
	return nil
}

// SubmitBacktestCommand 提交回测命令
type SubmitBacktestCommand struct {
	UserID     uint64
	Type       domain.StrategyType
	Symbol     string
	Parameters string
	StartTime  time.Time
	EndTime    time.Time
}

// SubmitBacktest 提交回测
func (s *CommandService) SubmitBacktest(ctx context.Context, cmd SubmitBacktestCommand) (string, error) {
	now := time.Now()
	backtestID := fmt.Sprintf("BT%s%04d", now.Format("20060102150405"), now.UnixNano()%10000)

	backtest := &domain.Backtest{
		BacktestID: backtestID,
		UserID:     cmd.UserID,
		Type:       cmd.Type,
		Symbol:     cmd.Symbol,
		Parameters: cmd.Parameters,
		StartTime:  cmd.StartTime,
		EndTime:    cmd.EndTime,
		Status:     "PENDING",
	}

	if err := s.backtestRepo.Save(ctx, backtest); err != nil {
		return "", err
	}

	s.logger.InfoContext(ctx, "backtest submitted", "backtest_id", backtestID)
	// 实际应发送任务到回测引擎消息队列
	return backtestID, nil
}

// publishEvents 发布领域事件
func (s *CommandService) publishEvents(ctx context.Context, events []domain.DomainEvent) {
	for _, event := range events {
		if err := s.eventPublisher.Publish(ctx, event.EventName(), "", event); err != nil {
			s.logger.ErrorContext(ctx, "failed to publish event",
				"event", event.EventName(),
				"error", err)
		}
	}
}
