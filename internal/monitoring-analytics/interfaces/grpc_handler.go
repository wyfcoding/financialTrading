package interfaces

import (
	"context"

	pb "github.com/fynnwu/FinancialTrading/go-api/monitoring-analytics"
	"github.com/fynnwu/FinancialTrading/internal/monitoring-analytics/application"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler gRPC 处理器
type GRPCHandler struct {
	pb.UnimplementedMonitoringAnalyticsServiceServer
	app *application.MonitoringAnalyticsService
}

// NewGRPCHandler 创建 gRPC 处理器实例
func NewGRPCHandler(app *application.MonitoringAnalyticsService) *GRPCHandler {
	return &GRPCHandler{app: app}
}

// RecordMetric 记录指标
func (h *GRPCHandler) RecordMetric(ctx context.Context, req *pb.RecordMetricRequest) (*pb.RecordMetricResponse, error) {
	err := h.app.RecordMetric(ctx, req.Metric.Name, req.Metric.Value, req.Metric.Tags, req.Metric.Timestamp.AsTime())
	if err != nil {
		return nil, err
	}

	return &pb.RecordMetricResponse{
		Success: true,
	}, nil
}

// GetMetrics 获取指标
func (h *GRPCHandler) GetMetrics(ctx context.Context, req *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	metrics, err := h.app.GetMetrics(ctx, req.Name, req.StartTime.AsTime(), req.EndTime.AsTime())
	if err != nil {
		return nil, err
	}

	protoMetrics := make([]*pb.Metric, len(metrics))
	for i, m := range metrics {
		protoMetrics[i] = &pb.Metric{
			Name:      m.Name,
			Value:     m.Value,
			Tags:      m.Tags,
			Timestamp: timestamppb.New(m.Timestamp),
		}
	}

	return &pb.GetMetricsResponse{
		Metrics: protoMetrics,
	}, nil
}

// GetSystemHealth 获取系统健康状态
func (h *GRPCHandler) GetSystemHealth(ctx context.Context, req *pb.GetSystemHealthRequest) (*pb.GetSystemHealthResponse, error) {
	healths, err := h.app.GetSystemHealth(ctx, req.ServiceName)
	if err != nil {
		return nil, err
	}

	protoHealths := make([]*pb.SystemHealth, len(healths))
	for i, h := range healths {
		protoHealths[i] = &pb.SystemHealth{
			ServiceName: h.ServiceName,
			Status:      h.Status,
			Message:     h.Message,
			LastChecked: timestamppb.New(h.LastChecked),
		}
	}

	return &pb.GetSystemHealthResponse{
		HealthStatuses: protoHealths,
	}, nil
}
