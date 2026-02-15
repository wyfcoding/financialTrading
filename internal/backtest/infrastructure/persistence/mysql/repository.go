// 生成摘要：实现回测服务的 MySQL 仓储层，基于 GORM。
// 变更说明：从旧的 infrastructure 目录迁移至 persistence/mysql。

package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/backtest/domain"
	"gorm.io/gorm"
)

type backtestRepository struct {
	db *gorm.DB
}

// NewBacktestRepository 创建回测仓储
func NewBacktestRepository(db *gorm.DB) domain.BacktestRepository {
	return &backtestRepository{db: db}
}

func (r *backtestRepository) SaveTask(ctx context.Context, task *domain.BacktestTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *backtestRepository) FindTaskByID(ctx context.Context, taskID string) (*domain.BacktestTask, error) {
	var task domain.BacktestTask
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *backtestRepository) SaveReport(ctx context.Context, report *domain.BacktestReport) error {
	return r.db.WithContext(ctx).Save(report).Error
}

func (r *backtestRepository) FindReportByTaskID(ctx context.Context, taskID string) (*domain.BacktestReport, error) {
	var report domain.BacktestReport
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).First(&report).Error; err != nil {
		return nil, err
	}
	return &report, nil
}
