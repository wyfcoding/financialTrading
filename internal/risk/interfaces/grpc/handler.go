// 包  gRPC 处理器实现
package grpc

import (
	"context"
	"fmt"
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

// CheckRisk 检查交易风险 (Legacy, mapped to AssessRisk)
func (h *Handler) CheckRisk(ctx context.Context, req *pb.CheckRiskRequest) (*pb.CheckRiskResponse, error) {
	dto, err := h.service.AssessRisk(ctx, &application.AssessRiskRequest{
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Side:     "buy", // Default side for legacy
		Quantity: fmt.Sprintf("%.8f", req.Quantity),
		Price:    fmt.Sprintf("%.8f", req.Price),
	})
	if err != nil {
		return &pb.CheckRiskResponse{Passed: false, Reason: err.Error()}, nil
	}
	return &pb.CheckRiskResponse{
		Passed: dto.IsAllowed,
		Reason: dto.Reason,
	}, nil
}

// SetRiskLimit 设置风险限额 (Legacy)
func (h *Handler) SetRiskLimit(ctx context.Context, req *pb.SetRiskLimitRequest) (*pb.SetRiskLimitResponse, error) {
	err := h.service.SetRiskLimit(ctx, req.UserId, "ORDER_SIZE", req.MaxOrderSize)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.SetRiskLimitResponse{Success: true}, nil
}

// AssessRisk 评估交易风险 (Enhanced)
func (h *Handler) AssessRisk(ctx context.Context, req *pb.AssessRiskRequest) (*pb.AssessRiskResponse, error) {
	start := time.Now()
	slog.Info("gRPC AssessRisk received", "user_id", req.UserId, "symbol", req.Symbol, "side", req.Side)

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

	return &pb.AssessRiskResponse{
		RiskLevel:         dto.RiskLevel,
		RiskScore:         dto.RiskScore,
		MarginRequirement: dto.MarginRequirement,
		IsAllowed:         dto.IsAllowed,
		Reason:            dto.Reason,
	}, nil
}

// GetRiskMetrics 获取风险指标
func (h *Handler) GetRiskMetrics(ctx context.Context, req *pb.GetRiskMetricsRequest) (*pb.GetRiskMetricsResponse, error) {
	metrics, err := h.service.GetRiskMetrics(ctx, req.UserId)
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
func (h *Handler) CheckRiskLimit(ctx context.Context, req *pb.CheckRiskLimitRequest) (*pb.CheckRiskLimitResponse, error) {
	limit, err := h.service.CheckRiskLimit(ctx, req.UserId, req.LimitType)
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
func (h *Handler) GetRiskAlerts(ctx context.Context, req *pb.GetRiskAlertsRequest) (*pb.GetRiskAlertsResponse, error) {
	alerts, err := h.service.GetRiskAlerts(ctx, req.UserId, 100)
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
