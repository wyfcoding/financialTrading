// 包  gRPC 处理器实现
package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/goapi/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/risk/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler gRPC 处理器
// 负责处理与风险管理相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedRiskServiceServer
	appService *application.RiskApplicationService // 风险应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// appService: 注入的风险应用服务
func NewGRPCHandler(appService *application.RiskApplicationService) *GRPCHandler {
	return &GRPCHandler{
		appService: appService,
	}
}

// AssessRisk 评估交易风险
// 处理 gRPC AssessRisk 请求
func (h *GRPCHandler) AssessRisk(ctx context.Context, req *pb.AssessRiskRequest) (*pb.AssessRiskResponse, error) {
	// 调用应用服务评估风险
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

	return &pb.AssessRiskResponse{
		RiskLevel:         dto.RiskLevel,
		RiskScore:         dto.RiskScore,
		MarginRequirement: dto.MarginRequirement,
		IsAllowed:         dto.IsAllowed,
		Reason:            dto.Reason,
	}, nil
}

// GetRiskMetrics 获取风险指标
// 处理 gRPC GetRiskMetrics 请求
func (h *GRPCHandler) GetRiskMetrics(ctx context.Context, req *pb.GetRiskMetricsRequest) (*pb.GetRiskMetricsResponse, error) {
	metrics, err := h.appService.GetRiskMetrics(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get risk metrics: %v", err)
	}

	return &pb.GetRiskMetricsResponse{
		Metrics: &pb.RiskMetrics{
			Var_95:      metrics.VaR95.String(),
			Var_99:      metrics.VaR99.String(),
			MaxDrawdown: metrics.MaxDrawdown.String(),
			SharpeRatio: metrics.SharpeRatio.String(),
			Correlation: metrics.Correlation.String(),
		},
	}, nil
}

// CheckRiskLimit 检查风险限额
// 处理 gRPC CheckRiskLimit 请求
func (h *GRPCHandler) CheckRiskLimit(ctx context.Context, req *pb.CheckRiskLimitRequest) (*pb.CheckRiskLimitResponse, error) {
	limit, err := h.appService.CheckRiskLimit(ctx, req.UserId, req.LimitType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check risk limit: %v", err)
	}

	remaining := limit.LimitValue.Sub(limit.CurrentValue)

	return &pb.CheckRiskLimitResponse{
		LimitType:    limit.LimitType,
		LimitValue:   limit.LimitValue.String(),
		CurrentValue: limit.CurrentValue.String(),
		Remaining:    remaining.String(),
		IsExceeded:   limit.IsExceeded,
	}, nil
}

// GetRiskAlerts 获取风险告警
// 处理 gRPC GetRiskAlerts 请求
func (h *GRPCHandler) GetRiskAlerts(ctx context.Context, req *pb.GetRiskAlertsRequest) (*pb.GetRiskAlertsResponse, error) {
	alerts, err := h.appService.GetRiskAlerts(ctx, req.UserId, 100) // Default limit
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get risk alerts: %v", err)
	}

	pbAlerts := make([]*pb.RiskAlert, 0, len(alerts))
	for _, alert := range alerts {
		pbAlerts = append(pbAlerts, &pb.RiskAlert{
			AlertId:   alert.ID,
			AlertType: alert.AlertType,
			Severity:  alert.Severity,
			Message:   alert.Message,
			Timestamp: alert.CreatedAt.Unix(),
		})
	}

	return &pb.GetRiskAlertsResponse{
		Alerts: pbAlerts,
	}, nil
}
