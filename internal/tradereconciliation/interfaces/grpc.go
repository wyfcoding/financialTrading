package interfaces

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/tradereconciliation/v1"
	"github.com/wyfcoding/financialtrading/internal/tradereconciliation/application"
	"github.com/wyfcoding/financialtrading/internal/tradereconciliation/domain"
)

type ReconciliationHandler struct {
	pb.UnimplementedTradeReconciliationServiceServer
	app  *application.ReconciliationService
	repo domain.ReconciliationRepository
}

func NewReconciliationHandler(app *application.ReconciliationService, repo domain.ReconciliationRepository) *ReconciliationHandler {
	return &ReconciliationHandler{app: app, repo: repo}
}

func (h *ReconciliationHandler) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	id, err := h.app.CreateTask(ctx, req.SourceA, req.SourceB, req.StartTime.AsTime(), req.EndTime.AsTime())
	if err != nil {
		return nil, err
	}
	return &pb.CreateTaskResponse{TaskId: id}, nil
}

func (h *ReconciliationHandler) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	task, err := h.app.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}
	return &pb.GetTaskResponse{
		TaskId:           task.TaskID,
		Status:           task.Status.String(),
		ProcessedCount:   task.ProcessedCount,
		DiscrepancyCount: task.DiscrepancyCount,
		CreatedAt:        task.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (h *ReconciliationHandler) ListDiscrepancies(ctx context.Context, req *pb.ListDiscrepanciesRequest) (*pb.ListDiscrepanciesResponse, error) {
	list, err := h.app.ListDiscrepancies(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}

	var res []*pb.DiscrepancyItem
	for _, d := range list {
		res = append(res, &pb.DiscrepancyItem{
			DiscrepancyId: d.DiscrepancyID,
			RecordId:      d.RecordID,
			Field:         d.Field,
			ValueA:        d.ValueA,
			ValueB:        d.ValueB,
			Status:        d.Status.String(),
		})
	}
	return &pb.ListDiscrepanciesResponse{Discrepancies: res}, nil
}

func (h *ReconciliationHandler) ResolveDiscrepancy(ctx context.Context, req *pb.ResolveDiscrepancyRequest) (*pb.ResolveDiscrepancyResponse, error) {
	err := h.app.ResolveDiscrepancy(ctx, req.DiscrepancyId, req.Resolution, req.Comment)
	if err != nil {
		return nil, err
	}
	return &pb.ResolveDiscrepancyResponse{Success: true}, nil
}
