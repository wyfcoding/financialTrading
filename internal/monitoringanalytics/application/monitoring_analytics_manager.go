package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
)

// MonitoringAnalyticsManager 处理所有监控和分析相关的写入操作（Commands）。
type MonitoringAnalyticsManager struct {
	metricRepo domain.MetricRepository
	healthRepo domain.SystemHealthRepository
}

// NewMonitoringAnalyticsManager 构造函数。
func NewMonitoringAnalyticsManager(metricRepo domain.MetricRepository, healthRepo domain.SystemHealthRepository) *MonitoringAnalyticsManager {
	return &MonitoringAnalyticsManager{
		metricRepo: metricRepo,
		healthRepo: healthRepo,
	}
}

// RecordMetric 记录指标
func (m *MonitoringAnalyticsManager) RecordMetric(ctx context.Context, name string, value decimal.Decimal, tags map[string]string, timestamp int64) error {
	metric := &domain.Metric{
		Name:      name,
		Value:     value,
		Tags:      tags,
		Timestamp: timestamp,
	}
	return m.metricRepo.Save(ctx, metric)
}

// SaveSystemHealth 保存系统健康状态
func (m *MonitoringAnalyticsManager) SaveSystemHealth(ctx context.Context, health *domain.SystemHealth) error {
	return m.healthRepo.Save(ctx, health)
}
