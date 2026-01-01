package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
)

// MonitoringAnalyticsQuery 处理所有监控和分析相关的查询操作（Queries）。
type MonitoringAnalyticsQuery struct {
	metricRepo domain.MetricRepository
	healthRepo domain.SystemHealthRepository
}

// NewMonitoringAnalyticsQuery 构造函数。
func NewMonitoringAnalyticsQuery(metricRepo domain.MetricRepository, healthRepo domain.SystemHealthRepository) *MonitoringAnalyticsQuery {
	return &MonitoringAnalyticsQuery{
		metricRepo: metricRepo,
		healthRepo: healthRepo,
	}
}

// GetMetrics 获取指标历史数据
func (q *MonitoringAnalyticsQuery) GetMetrics(ctx context.Context, name string, startTime, endTime int64) ([]*domain.Metric, error) {
	return q.metricRepo.GetMetrics(ctx, name, startTime, endTime)
}

// GetSystemHealth 获取服务最新的健康检查记录
func (q *MonitoringAnalyticsQuery) GetSystemHealth(ctx context.Context, serviceName string) ([]*domain.SystemHealth, error) {
	return q.healthRepo.GetLatestHealth(ctx, serviceName, 10)
}
