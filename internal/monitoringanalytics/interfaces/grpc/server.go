package grpc

import (
	"context"
	"time"

	v1 "github.com/wyfcoding/financialtrading/go-api/monitoringanalytics/v1"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedMonitoringAnalyticsServer
	app *application.MonitoringAnalyticsService
}

func NewServer(s *grpc.Server, app *application.MonitoringAnalyticsService) *Server {
	srv := &Server{app: app}
	v1.RegisterMonitoringAnalyticsServer(s, srv)
	return srv
}

func (s *Server) GetMetrics(ctx context.Context, req *v1.GetMetricsRequest) (*v1.GetMetricsResponse, error) {
	// Default time range: last 24h
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	dtos, err := s.app.GetTradeMetrics(ctx, req.Symbol, startTime, endTime)
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

func (s *Server) GetAlerts(ctx context.Context, req *v1.GetAlertsRequest) (*v1.GetAlertsResponse, error) {
	dtos, err := s.app.GetAlerts(ctx, int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &v1.GetAlertsResponse{
		Alerts: make([]*v1.Alert, len(dtos)),
	}

	for i, d := range dtos {
		resp.Alerts[i] = &v1.Alert{
			Id:        string(d.ID),
			Severity:  d.Severity,
			Message:   d.Message,
			Timestamp: timestamppb.New(d.Timestamp),
		}
	}
	return resp, nil
}
