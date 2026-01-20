package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
)

// MonitoringAnalyticsManager 处理监控相关的写入操作
type MonitoringAnalyticsManager struct {
	metricRepo domain.MetricRepository
	healthRepo domain.SystemHealthRepository
}

func NewMonitoringAnalyticsManager(metricRepo domain.MetricRepository, healthRepo domain.SystemHealthRepository) *MonitoringAnalyticsManager {
	return &MonitoringAnalyticsManager{
		metricRepo: metricRepo,
		healthRepo: healthRepo,
	}
}

func (m *MonitoringAnalyticsManager) RecordMetric(ctx context.Context, name string, value decimal.Decimal, tags map[string]string, timestamp int64) error {
	metric := &domain.Metric{
		Name:      name,
		Value:     value,
		Tags:      tags,
		Timestamp: timestamp,
	}
	return m.metricRepo.Save(ctx, metric)
}

func (m *MonitoringAnalyticsManager) SaveSystemHealth(ctx context.Context, health *domain.SystemHealth) error {
	return m.healthRepo.Save(ctx, health)
}
