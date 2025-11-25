package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wyfcoding/financialTrading/internal/monitoring-analytics/domain"
	"github.com/wyfcoding/financialTrading/pkg/logger"
)

// MonitoringAnalyticsService 应用服务
type MonitoringAnalyticsService struct {
	metricRepo domain.MetricRepository
	healthRepo domain.SystemHealthRepository
}

// NewMonitoringAnalyticsService 创建应用服务实例
func NewMonitoringAnalyticsService(metricRepo domain.MetricRepository, healthRepo domain.SystemHealthRepository) *MonitoringAnalyticsService {
	return &MonitoringAnalyticsService{
		metricRepo: metricRepo,
		healthRepo: healthRepo,
	}
}

// RecordMetric 记录指标
func (s *MonitoringAnalyticsService) RecordMetric(ctx context.Context, name string, value float64, tags map[string]string, timestamp time.Time) error {
	metric := &domain.Metric{
		Name:      name,
		Value:     value,
		Tags:      tags,
		Timestamp: timestamp,
	}
	if err := s.metricRepo.Save(ctx, metric); err != nil {
		logger.Error(ctx, "Failed to save metric",
			"name", name,
			"error", err,
		)
		return fmt.Errorf("failed to save metric: %w", err)
	}

	// Optional: Log metric recording at debug level to avoid spamming
	logger.Debug(ctx, "Metric recorded",
		"name", name,
		"value", value,
	)

	return nil
}

// GetMetrics 获取指标
func (s *MonitoringAnalyticsService) GetMetrics(ctx context.Context, name string, startTime, endTime time.Time) ([]*domain.Metric, error) {
	metrics, err := s.metricRepo.GetMetrics(ctx, name, startTime, endTime)
	if err != nil {
		logger.Error(ctx, "Failed to get metrics",
			"name", name,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}
	return metrics, nil
}

// GetSystemHealth 获取系统健康状态
func (s *MonitoringAnalyticsService) GetSystemHealth(ctx context.Context, serviceName string) ([]*domain.SystemHealth, error) {
	healths, err := s.healthRepo.GetLatestHealth(ctx, serviceName)
	if err != nil {
		logger.Error(ctx, "Failed to get system health",
			"service_name", serviceName,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get system health: %w", err)
	}
	return healths, nil
}
