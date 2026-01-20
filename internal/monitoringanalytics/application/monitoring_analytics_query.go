package application

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
)

// MonitoringAnalyticsQuery 处理所有监控和分析相关的查询操作（Queries）。
type MonitoringAnalyticsQuery struct {
	metricRepo domain.MetricRepository
	healthRepo domain.SystemHealthRepository
	alertRepo  domain.AlertRepository
}

// NewMonitoringAnalyticsQuery 构造函数。
func NewMonitoringAnalyticsQuery(metricRepo domain.MetricRepository, healthRepo domain.SystemHealthRepository, alertRepo domain.AlertRepository) *MonitoringAnalyticsQuery {
	return &MonitoringAnalyticsQuery{
		metricRepo: metricRepo,
		healthRepo: healthRepo,
		alertRepo:  alertRepo,
	}
}

// GetMetrics 获取指标历史数据
func (q *MonitoringAnalyticsQuery) GetMetrics(ctx context.Context, name string, startTime, endTime int64) ([]*domain.Metric, error) {
	return q.metricRepo.GetMetrics(ctx, name, startTime, endTime)
}

func (q *MonitoringAnalyticsQuery) GetTradeMetrics(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*domain.TradeMetric, error) {
	return q.metricRepo.GetTradeMetrics(ctx, symbol, startTime, endTime)
}

func (q *MonitoringAnalyticsQuery) GetSystemHealth(ctx context.Context, serviceName string) ([]*domain.SystemHealth, error) {
	return q.healthRepo.GetLatestHealth(ctx, serviceName, 10)
}

func (q *MonitoringAnalyticsQuery) GetAlerts(ctx context.Context, limit int) ([]*domain.Alert, error) {
	return q.alertRepo.GetAlerts(ctx, limit)
}
