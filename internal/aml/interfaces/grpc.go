package interfaces

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/aml/v1"
	"github.com/wyfcoding/financialtrading/internal/aml/application"
)

type AMLHandler struct {
	pb.UnimplementedAMLServiceServer
	app *application.AMLService
}

func NewAMLHandler(app *application.AMLService) *AMLHandler {
	return &AMLHandler{app: app}
}

func (h *AMLHandler) MonitorTransaction(ctx context.Context, req *pb.MonitorTransactionRequest) (*pb.MonitorTransactionResponse, error) {
	return h.app.MonitorTransaction(ctx, req)
}

func (h *AMLHandler) GetRiskScore(ctx context.Context, req *pb.GetRiskScoreRequest) (*pb.GetRiskScoreResponse, error) {
	return h.app.GetRiskScore(ctx, req.UserId)
}

func (h *AMLHandler) ListAlerts(ctx context.Context, req *pb.ListAlertsRequest) (*pb.ListAlertsResponse, error) {
	return h.app.ListAlerts(ctx, req.Status)
}
