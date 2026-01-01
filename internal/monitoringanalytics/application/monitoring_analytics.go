package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
)

// MonitoringAnalyticsService 监控分析门面服务，整合 Manager 和 Query。
type MonitoringAnalyticsService struct {
	manager *MonitoringAnalyticsManager
	query   *MonitoringAnalyticsQuery
}

// NewMonitoringAnalyticsService 构造函数。
func NewMonitoringAnalyticsService(metricRepo domain.MetricRepository, healthRepo domain.SystemHealthRepository) *MonitoringAnalyticsService {
	return &MonitoringAnalyticsService{
		manager: NewMonitoringAnalyticsManager(metricRepo, healthRepo),
		query:   NewMonitoringAnalyticsQuery(metricRepo, healthRepo),
	}
}

// --- Manager (Writes) ---

func (s *MonitoringAnalyticsService) RecordMetric(ctx context.Context, name string, value decimal.Decimal, tags map[string]string, timestamp int64) error {
	return s.manager.RecordMetric(ctx, name, value, tags, timestamp)
}

func (s *MonitoringAnalyticsService) SaveSystemHealth(ctx context.Context, health *domain.SystemHealth) error {
	return s.manager.SaveSystemHealth(ctx, health)
}

// --- Query (Reads) ---

func (s *MonitoringAnalyticsService) GetMetrics(ctx context.Context, name string, startTime, endTime int64) ([]*domain.Metric, error) {
	return s.query.GetMetrics(ctx, name, startTime, endTime)
}

func (s *MonitoringAnalyticsService) GetSystemHealth(ctx context.Context, serviceName string) ([]*domain.SystemHealth, error) {
	return s.query.GetSystemHealth(ctx, serviceName)
}
