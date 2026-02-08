package infrastructure

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/tradereconciliation/domain"
	"gorm.io/gorm"
)

type ReconciliationRepository struct {
	db *gorm.DB
}

func NewReconciliationRepository(db *gorm.DB) *ReconciliationRepository {
	return &ReconciliationRepository{db: db}
}

func (r *ReconciliationRepository) SaveTask(ctx context.Context, task *domain.ReconciliationTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *ReconciliationRepository) GetTask(ctx context.Context, taskID string) (*domain.ReconciliationTask, error) {
	var t domain.ReconciliationTask
	if err := r.db.WithContext(ctx).Preload("Discrepancies").Where("task_id = ?", taskID).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ReconciliationRepository) SaveDiscrepancy(ctx context.Context, d *domain.Discrepancy) error {
	return r.db.WithContext(ctx).Save(d).Error
}

func (r *ReconciliationRepository) GetDiscrepancy(ctx context.Context, id string) (*domain.Discrepancy, error) {
	var d domain.Discrepancy
	if err := r.db.WithContext(ctx).Where("discrepancy_id = ?", id).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *ReconciliationRepository) ListDiscrepancies(ctx context.Context, taskID string) ([]domain.Discrepancy, error) {
	var ds []domain.Discrepancy
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).Find(&ds).Error; err != nil {
		return nil, err
	}
	return ds, nil
}
