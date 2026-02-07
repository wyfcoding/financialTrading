package grpc

import (
	"context"
	"time"

	v1 "github.com/wyfcoding/financialtrading/go-api/monitoringanalytics/v1"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	v1.UnimplementedMonitoringAnalyticsServer
	query *application.MonitoringAnalyticsQueryService
}

func NewHandler(query *application.MonitoringAnalyticsQueryService) *Handler {
	return &Handler{query: query}
}

func (h *Handler) GetMetrics(ctx context.Context, req *v1.GetMetricsRequest) (*v1.GetMetricsResponse, error) {
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)
	if req.TimeRange != "" {
		if dur, err := time.ParseDuration(req.TimeRange); err == nil {
			startTime = endTime.Add(-dur)
		} else if len(req.TimeRange) > 1 && req.TimeRange[len(req.TimeRange)-1] == 'd' {
			if days, err := time.ParseDuration(req.TimeRange[:len(req.TimeRange)-1] + "24h"); err == nil {
				startTime = endTime.Add(-days)
			}
		}
	}

	dtos, err := h.query.GetTradeMetrics(ctx, req.Symbol, startTime, endTime)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &v1.GetMetricsResponse{
		Points: make([]*v1.MetricPoint, len(dtos)),
	}

	for i, d := range dtos {
		resp.Points[i] = &v1.MetricPoint{
			Timestamp:    timestamppb.New(d.Timestamp),
			Volume:       d.TotalVolume,
			TradeCount:   float64(d.TradeCount),
			AveragePrice: d.AveragePrice,
		}
	}

	return resp, nil
}

func (h *Handler) GetAlerts(ctx context.Context, req *v1.GetAlertsRequest) (*v1.GetAlertsResponse, error) {
	dtos, err := h.query.GetAlerts(ctx, int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &v1.GetAlertsResponse{
		Alerts: make([]*v1.Alert, len(dtos)),
	}

	for i, d := range dtos {
		resp.Alerts[i] = &v1.Alert{
			Id:        d.AlertID,
			Severity:  d.Severity,
			Message:   d.Message,
			Timestamp: timestamppb.New(time.Unix(d.Timestamp(), 0)),
		}
	}
	return resp, nil
}
