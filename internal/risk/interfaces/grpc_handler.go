package interfaces

import (
	"context"

	pb "github.com/fynnwu/FinancialTrading/go-api/risk"
	"github.com/fynnwu/FinancialTrading/internal/risk/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	pb.UnimplementedRiskServiceServer
	appService *application.RiskApplicationService
}

func NewGRPCHandler(appService *application.RiskApplicationService) *GRPCHandler {
	return &GRPCHandler{
		appService: appService,
	}
}

func (h *GRPCHandler) AssessRisk(ctx context.Context, req *pb.AssessRiskRequest) (*pb.RiskAssessmentResponse, error) {
	dto, err := h.appService.AssessRisk(ctx, &application.AssessRiskRequest{
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Side:     req.Side,
		Quantity: req.Quantity,
		Price:    req.Price,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to assess risk: %v", err)
	}

	return &pb.RiskAssessmentResponse{
		RiskLevel:         dto.RiskLevel,
		RiskScore:         dto.RiskScore,
		MarginRequirement: dto.MarginRequirement,
		IsAllowed:         dto.IsAllowed,
		Reason:            dto.Reason,
	}, nil
}

func (h *GRPCHandler) GetRiskMetrics(ctx context.Context, req *pb.GetRiskMetricsRequest) (*pb.RiskMetricsResponse, error) {
	metrics, err := h.appService.GetRiskMetrics(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get risk metrics: %v", err)
	}

	return &pb.RiskMetricsResponse{
		Var_95:      metrics.VaR95.String(),
		Var_99:      metrics.VaR99.String(),
		MaxDrawdown: metrics.MaxDrawdown.String(),
		SharpeRatio: metrics.SharpeRatio.String(),
		Correlation: metrics.Correlation.String(),
	}, nil
}

func (h *GRPCHandler) CheckRiskLimit(ctx context.Context, req *pb.CheckRiskLimitRequest) (*pb.RiskLimitResponse, error) {
	limit, err := h.appService.CheckRiskLimit(ctx, req.UserId, req.LimitType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check risk limit: %v", err)
	}

	remaining := limit.LimitValue.Sub(limit.CurrentValue)

	return &pb.RiskLimitResponse{
		LimitType:    limit.LimitType,
		LimitValue:   limit.LimitValue.String(),
		CurrentValue: limit.CurrentValue.String(),
		Remaining:    remaining.String(),
		IsExceeded:   limit.IsExceeded,
	}, nil
}

func (h *GRPCHandler) GetRiskAlerts(ctx context.Context, req *pb.GetRiskAlertsRequest) (*pb.RiskAlertsResponse, error) {
	alerts, err := h.appService.GetRiskAlerts(ctx, req.UserId, 100) // Default limit
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get risk alerts: %v", err)
	}

	pbAlerts := make([]*pb.RiskAlert, 0, len(alerts))
	for _, alert := range alerts {
		pbAlerts = append(pbAlerts, &pb.RiskAlert{
			AlertId:   alert.AlertID,
			AlertType: alert.AlertType,
			Severity:  alert.Severity,
			Message:   alert.Message,
			Timestamp: alert.CreatedAt.Unix(),
		})
	}

	return &pb.RiskAlertsResponse{
		Alerts: pbAlerts,
	}, nil
}
