package domain

import (
	"context"
)

type ReconciliationRepository interface {
	SaveTask(ctx context.Context, task *ReconciliationTask) error
	GetTask(ctx context.Context, taskID string) (*ReconciliationTask, error)

	SaveDiscrepancy(ctx context.Context, d *Discrepancy) error
	GetDiscrepancy(ctx context.Context, id string) (*Discrepancy, error)
	ListDiscrepancies(ctx context.Context, taskID string) ([]Discrepancy, error)
}
