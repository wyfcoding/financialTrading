// 包 监控分析服务的用例逻辑
package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
	"github.com/wyfcoding/pkg/logging"
)

// MonitoringAnalyticsService 监控分析应用服务
// 负责收集、存储和查询系统监控指标及健康状态
type MonitoringAnalyticsService struct {
	metricRepo domain.MetricRepository       // 指标仓储接口
	healthRepo domain.SystemHealthRepository // 健康状态仓储接口
}

// NewMonitoringAnalyticsService 创建监控分析应用服务实例
// metricRepo: 注入的指标仓储实现
// healthRepo: 注入的健康状态仓储实现
func NewMonitoringAnalyticsService(metricRepo domain.MetricRepository, healthRepo domain.SystemHealthRepository) *MonitoringAnalyticsService {
	return &MonitoringAnalyticsService{
		metricRepo: metricRepo,
		healthRepo: healthRepo,
	}
}

// RecordMetric 记录指标
func (s *MonitoringAnalyticsService) RecordMetric(ctx context.Context, name string, value decimal.Decimal, tags map[string]string, timestamp int64) error {
	metric := &domain.Metric{
		Name:      name,
		Value:     value,
		Tags:      tags,
		Timestamp: timestamp,
	}
	if err := s.metricRepo.Save(ctx, metric); err != nil {
		logging.Error(ctx, "Failed to save metric",
			"name", name,
			"error", err,
		)
		return fmt.Errorf("failed to save metric: %w", err)
	}

	logging.Debug(ctx, "Metric recorded",
		"name", name,
		"value", value.String(),
	)

	return nil
}

// GetMetrics 获取指标历史数据
func (s *MonitoringAnalyticsService) GetMetrics(ctx context.Context, name string, startTime, endTime int64) ([]*domain.Metric, error) {
	metrics, err := s.metricRepo.GetMetrics(ctx, name, startTime, endTime)
	if err != nil {
		logging.Error(ctx, "Failed to get metrics",
			"name", name,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}
	return metrics, nil
}

// GetSystemHealth 获取服务最新的健康检查记录
func (s *MonitoringAnalyticsService) GetSystemHealth(ctx context.Context, serviceName string) ([]*domain.SystemHealth, error) {
	// 默认获取最近 10 条记录
	healths, err := s.healthRepo.GetLatestHealth(ctx, serviceName, 10)
	if err != nil {
		logging.Error(ctx, "Failed to get system health",
			"service_name", serviceName,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get system health: %w", err)
	}
	return healths, nil
}
