package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/backtest/domain"
)

// RunBacktestCommand 运行回测命令
type RunBacktestCommand struct {
	StrategyID     string
	Symbol         string
	StartTime      time.Time
	EndTime        time.Time
	InitialCapital float64
}

// BacktestApplicationService 回测应用服务
type BacktestApplicationService struct {
	engine *domain.BacktestEngine
	repo   domain.BacktestRepository
	logger *slog.Logger
}

func NewBacktestApplicationService(engine *domain.BacktestEngine, repo domain.BacktestRepository, logger *slog.Logger) *BacktestApplicationService {
	return &BacktestApplicationService{
		engine: engine,
		repo:   repo,
		logger: logger,
	}
}

func (s *BacktestApplicationService) RunBacktest(ctx context.Context, cmd RunBacktestCommand) (string, error) {
	taskID := fmt.Sprintf("BT-%d", time.Now().UnixNano())
	s.logger.Info("starting backtest task", "task_id", taskID, "strategy", cmd.StrategyID)

	task := &domain.BacktestTask{
		TaskID:         taskID,
		StrategyID:     cmd.StrategyID,
		Symbol:         cmd.Symbol,
		StartTime:      cmd.StartTime,
		EndTime:        cmd.EndTime,
		InitialCapital: cmd.InitialCapital,
		Status:         "PENDING",
	}

	if err := s.repo.SaveTask(ctx, task); err != nil {
		return "", err
	}

	// 异步运行回测
	go func() {
		report, err := s.engine.Run(context.Background(), task)
		if err != nil {
			s.logger.Error("backtest failed", "task_id", taskID, "error", err)
			task.Status = "FAILED"
			s.repo.SaveTask(context.Background(), task)
			return
		}

		task.Status = "COMPLETED"
		s.repo.SaveTask(context.Background(), task)
		s.repo.SaveReport(context.Background(), report)
		s.logger.Info("backtest completed", "task_id", taskID, "return", report.TotalReturn)
	}()

	return taskID, nil
}

func (s *BacktestApplicationService) GetReport(ctx context.Context, taskID string) (*domain.BacktestReport, error) {
	return s.repo.FindReportByTaskID(ctx, taskID)
}
