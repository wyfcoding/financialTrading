package domain

import (
	"context"
)

// BacktestRepository 回测仓储接口
type BacktestRepository interface {
	SaveTask(ctx context.Context, task *BacktestTask) error
	FindTaskByID(ctx context.Context, taskID string) (*BacktestTask, error)
	SaveReport(ctx context.Context, report *BacktestReport) error
	FindReportByTaskID(ctx context.Context, taskID string) (*BacktestReport, error)
}
