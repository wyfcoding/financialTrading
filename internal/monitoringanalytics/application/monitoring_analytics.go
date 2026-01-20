package application

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
)

// MonitoringAnalyticsService 监控分析门面服务，整合 Manager 和 Query。
type MonitoringAnalyticsService struct {
	manager *MonitoringAnalyticsManager
	query   *MonitoringAnalyticsQuery
}

// NewMonitoringAnalyticsService 构造函数。
func NewMonitoringAnalyticsService(metricRepo domain.MetricRepository, healthRepo domain.SystemHealthRepository, alertRepo domain.AlertRepository) *MonitoringAnalyticsService {
	return &MonitoringAnalyticsService{
		manager: NewMonitoringAnalyticsManager(metricRepo, healthRepo),
		query:   NewMonitoringAnalyticsQuery(metricRepo, healthRepo, alertRepo),
	}
}

// ... (Manager methods unchanged)

// --- Query (Reads) ---

func (s *MonitoringAnalyticsService) GetMetrics(ctx context.Context, name string, startTime, endTime int64) ([]*domain.Metric, error) {
	return s.query.GetMetrics(ctx, name, startTime, endTime)
}

func (s *MonitoringAnalyticsService) GetTradeMetrics(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*domain.TradeMetric, error) {
	return s.query.GetTradeMetrics(ctx, symbol, startTime, endTime)
}

func (s *MonitoringAnalyticsService) GetSystemHealth(ctx context.Context, serviceName string) ([]*domain.SystemHealth, error) {
	return s.query.GetSystemHealth(ctx, serviceName)
}

func (s *MonitoringAnalyticsService) GetAlerts(ctx context.Context, limit int) ([]*domain.Alert, error) {
	return s.query.GetAlerts(ctx, limit)
}
