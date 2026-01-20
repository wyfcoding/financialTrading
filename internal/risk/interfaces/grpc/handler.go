// 包  gRPC 处理器实现
package grpc

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialtrading/go-api/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/risk/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler gRPC 处理器
// 负责处理与风险管理相关的 gRPC 请求
type Handler struct {
	pb.UnimplementedRiskServiceServer
	service *application.RiskService // 风险应用服务
}

// NewHandler 创建 gRPC 处理器实例
// service: 注入的风险应用服务
func NewHandler(service *application.RiskService) *Handler {
	return &Handler{
		service: service,
	}
}

// AssessRisk 评估交易风险
// 处理 gRPC AssessRisk 请求
func (h *Handler) AssessRisk(ctx context.Context, req *pb.AssessRiskRequest) (*pb.AssessRiskResponse, error) {
	start := time.Now()
	slog.Info("gRPC AssessRisk received", "user_id", req.UserId, "symbol", req.Symbol, "side", req.Side)

	// 调用应用服务评估风险
	dto, err := h.service.AssessRisk(ctx, &application.AssessRiskRequest{
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Side:     req.Side,
		Quantity: req.Quantity,
		Price:    req.Price,
	})
	if err != nil {
		slog.Error("gRPC AssessRisk failed", "user_id", req.UserId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to assess risk: %v", err)
	}

	slog.Info("gRPC AssessRisk successful", "user_id", req.UserId, "risk_level", dto.RiskLevel, "is_allowed", dto.IsAllowed, "duration", time.Since(start))
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
func (h *Handler) GetRiskMetrics(ctx context.Context, req *pb.GetRiskMetricsRequest) (*pb.GetRiskMetricsResponse, error) {
	start := time.Now()
	slog.Debug("gRPC GetRiskMetrics received", "user_id", req.UserId)

	metrics, err := h.service.GetRiskMetrics(ctx, req.UserId)
	if err != nil {
		slog.Error("gRPC GetRiskMetrics failed", "user_id", req.UserId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get risk metrics: %v", err)
	}

	slog.Debug("gRPC GetRiskMetrics successful", "user_id", req.UserId, "duration", time.Since(start))
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
func (h *Handler) CheckRiskLimit(ctx context.Context, req *pb.CheckRiskLimitRequest) (*pb.CheckRiskLimitResponse, error) {
	start := time.Now()
	slog.Debug("gRPC CheckRiskLimit received", "user_id", req.UserId, "limit_type", req.LimitType)

	limit, err := h.service.CheckRiskLimit(ctx, req.UserId, req.LimitType)
	if err != nil {
		slog.Error("gRPC CheckRiskLimit failed", "user_id", req.UserId, "limit_type", req.LimitType, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to check risk limit: %v", err)
	}

	remaining := limit.LimitValue.Sub(limit.CurrentValue)

	slog.Debug("gRPC CheckRiskLimit successful", "user_id", req.UserId, "limit_type", req.LimitType, "is_exceeded", limit.IsExceeded, "duration", time.Since(start))
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
func (h *Handler) GetRiskAlerts(ctx context.Context, req *pb.GetRiskAlertsRequest) (*pb.GetRiskAlertsResponse, error) {
	start := time.Now()
	slog.Debug("gRPC GetRiskAlerts received", "user_id", req.UserId)

	alerts, err := h.service.GetRiskAlerts(ctx, req.UserId, 100) // Default limit
	if err != nil {
		slog.Error("gRPC GetRiskAlerts failed", "user_id", req.UserId, "error", err, "duration", time.Since(start))
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

	slog.Debug("gRPC GetRiskAlerts successful", "user_id", req.UserId, "alerts_count", len(pbAlerts), "duration", time.Since(start))
	return &pb.GetRiskAlertsResponse{
		Alerts: pbAlerts,
	}, nil
}
