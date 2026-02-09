package infrastructure

import (
	"context"

	"github.com/wyfcoding/financialTrading/internal/backtest/domain"
	"gorm.io/gorm"
)

type BacktestRepositoryImpl struct {
	db *gorm.DB
}

func NewBacktestRepository(db *gorm.DB) domain.BacktestRepository {
	return &BacktestRepositoryImpl{db: db}
}

func (r *BacktestRepositoryImpl) SaveTask(ctx context.Context, task *domain.BacktestTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *BacktestRepositoryImpl) FindTaskByID(ctx context.Context, taskID string) (*domain.BacktestTask, error) {
	var task domain.BacktestTask
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *BacktestRepositoryImpl) SaveReport(ctx context.Context, report *domain.BacktestReport) error {
	return r.db.WithContext(ctx).Save(report).Error
}

func (r *BacktestRepositoryImpl) FindReportByTaskID(ctx context.Context, taskID string) (*domain.BacktestReport, error) {
	var report domain.BacktestReport
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).First(&report).Error; err != nil {
		return nil, err
	}
	return &report, nil
}
