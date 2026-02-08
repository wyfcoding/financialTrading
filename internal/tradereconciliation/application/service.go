package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wyfcoding/financialtrading/internal/tradereconciliation/domain"
)

type ReconciliationService struct {
	repo domain.ReconciliationRepository
}

func NewReconciliationService(repo domain.ReconciliationRepository) *ReconciliationService {
	return &ReconciliationService{repo: repo}
}

func (s *ReconciliationService) CreateTask(ctx context.Context, sourceA, sourceB string, start, end time.Time) (string, error) {
	id := fmt.Sprintf("TASK-%d", time.Now().UnixNano())
	task := domain.NewTask(id, sourceA, sourceB, start, end)

	if err := s.repo.SaveTask(ctx, task); err != nil {
		return "", err
	}

	// Async execution mock
	go s.runTask(id)

	return id, nil
}

func (s *ReconciliationService) runTask(taskID string) {
	// Simulate async processing
	ctx := context.Background()
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return
	}

	task.Start()
	_ = s.repo.SaveTask(ctx, task)

	time.Sleep(2 * time.Second) // Simulate work

	// Mock outcome: 1 discrepancy found
	dID := fmt.Sprintf("DISC-%d", time.Now().UnixNano())
	d := &domain.Discrepancy{
		DiscrepancyID: dID,
		TaskID:        taskID,
		RecordID:      "TRD-999",
		Field:         "Price",
		ValueA:        "100.00",
		ValueB:        "100.50",
		Status:        domain.DiscrepancyOpen,
	}
	_ = s.repo.SaveDiscrepancy(ctx, d)

	task.ProcessedCount = 100
	task.DiscrepancyCount = 1
	task.Complete()
	_ = s.repo.SaveTask(ctx, task)
}

func (s *ReconciliationService) GetTask(ctx context.Context, taskID string) (*domain.ReconciliationTask, error) {
	return s.repo.GetTask(ctx, taskID)
}

func (s *ReconciliationService) ListDiscrepancies(ctx context.Context, taskID string) ([]domain.Discrepancy, error) {
	return s.repo.ListDiscrepancies(ctx, taskID)
}

func (s *ReconciliationService) ResolveDiscrepancy(ctx context.Context, discrepancyID, resolution, comment string) error {
	d, err := s.repo.GetDiscrepancy(ctx, discrepancyID)
	if err != nil {
		return err
	}

	if err := d.Resolve(resolution, comment); err != nil {
		return err
	}

	return s.repo.SaveDiscrepancy(ctx, d)
}
