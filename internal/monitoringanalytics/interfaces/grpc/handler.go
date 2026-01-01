// 包  gRPC 处理器实现
package grpc

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/goapi/monitoringanalytics/v1"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/application"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler gRPC 处理器
// 负责处理与监控分析相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedMonitoringAnalyticsServiceServer
	app *application.MonitoringAnalyticsService // 监控分析应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// app: 注入的监控分析应用服务
func NewGRPCHandler(app *application.MonitoringAnalyticsService) *GRPCHandler {
	return &GRPCHandler{app: app}
}

// RecordMetric 记录指标
func (h *GRPCHandler) RecordMetric(ctx context.Context, req *pb.RecordMetricRequest) (*pb.RecordMetricResponse, error) {
	val := decimal.NewFromFloat(req.Metric.Value)
	ts := req.Metric.Timestamp.AsTime().UnixMilli()
	err := h.app.RecordMetric(ctx, req.Metric.Name, val, req.Metric.Tags, ts)
	if err != nil {
		return nil, err
	}

	return &pb.RecordMetricResponse{
		Success: true,
	}, nil
}

// GetMetrics 获取指标
func (h *GRPCHandler) GetMetrics(ctx context.Context, req *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	start := req.StartTime.AsTime().UnixMilli()
	end := req.EndTime.AsTime().UnixMilli()
	metrics, err := h.app.GetMetrics(ctx, req.Name, start, end)
	if err != nil {
		return nil, err
	}

	protoMetrics := make([]*pb.Metric, len(metrics))
	for i, m := range metrics {
		val, ok := m.Value.Float64()
		if !ok {
			// 记录日志或处理
		}
		protoMetrics[i] = &pb.Metric{
			Name:      m.Name,
			Value:     val,
			Tags:      m.Tags,
			Timestamp: timestamppb.New(time.UnixMilli(m.Timestamp)),
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
			LastChecked: timestamppb.New(time.UnixMilli(h.LastChecked)),
		}
	}

	return &pb.GetSystemHealthResponse{
		HealthStatuses: protoHealths,
	}, nil
}
