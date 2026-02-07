package application

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
)

// MonitoringAnalyticsQueryService 处理所有监控和分析相关的查询操作（Queries）。
type MonitoringAnalyticsQueryService struct {
	metricRepo     domain.MetricRepository
	metricReadRepo domain.MetricReadRepository
	healthRepo     domain.SystemHealthRepository
	healthReadRepo domain.SystemHealthReadRepository
	alertRepo      domain.AlertRepository
	alertReadRepo  domain.AlertReadRepository
	auditRepo      domain.ExecutionAuditRepository
	auditESRepo    domain.AuditESRepository
}

// NewMonitoringAnalyticsQueryService 构造函数。
func NewMonitoringAnalyticsQueryService(
	metricRepo domain.MetricRepository,
	metricReadRepo domain.MetricReadRepository,
	healthRepo domain.SystemHealthRepository,
	healthReadRepo domain.SystemHealthReadRepository,
	alertRepo domain.AlertRepository,
	alertReadRepo domain.AlertReadRepository,
	auditRepo domain.ExecutionAuditRepository,
	auditESRepo domain.AuditESRepository,
) *MonitoringAnalyticsQueryService {
	return &MonitoringAnalyticsQueryService{
		metricRepo:     metricRepo,
		metricReadRepo: metricReadRepo,
		healthRepo:     healthRepo,
		healthReadRepo: healthReadRepo,
		alertRepo:      alertRepo,
		alertReadRepo:  alertReadRepo,
		auditRepo:      auditRepo,
		auditESRepo:    auditESRepo,
	}
}

// SearchAudit 审计流水搜索 (Elasticsearch)
func (q *MonitoringAnalyticsQueryService) SearchAudit(ctx context.Context, query string, from, size int) ([]*domain.ExecutionAudit, int64, error) {
	if q.auditESRepo == nil {
		return nil, 0, nil
	}
	return q.auditESRepo.Search(ctx, query, from, size)
}

// QueryAudit 审计流水查询 (ClickHouse - 精确查询)
func (q *MonitoringAnalyticsQueryService) QueryAudit(ctx context.Context, userID, symbol string, startTime, endTime int64) ([]*domain.ExecutionAudit, error) {
	if q.auditRepo == nil {
		return nil, nil
	}
	return q.auditRepo.Query(ctx, userID, symbol, startTime, endTime)
}

// GetMetrics 获取指标历史数据
func (q *MonitoringAnalyticsQueryService) GetMetrics(ctx context.Context, name string, startTime, endTime int64) ([]*domain.Metric, error) {
	if q.metricReadRepo != nil {
		if cached, err := q.metricReadRepo.ListRecent(ctx, name, 1000); err == nil && len(cached) > 0 {
			filtered := make([]*domain.Metric, 0, len(cached))
			for _, m := range cached {
				if m == nil {
					continue
				}
				if m.Timestamp >= startTime && m.Timestamp <= endTime {
					filtered = append(filtered, m)
				}
			}
			if len(filtered) > 0 {
				return filtered, nil
			}
		}
	}
	return q.metricRepo.GetMetrics(ctx, name, startTime, endTime)
}

func (q *MonitoringAnalyticsQueryService) GetTradeMetrics(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*domain.TradeMetric, error) {
	return q.metricRepo.GetTradeMetrics(ctx, symbol, startTime, endTime)
}

func (q *MonitoringAnalyticsQueryService) GetSystemHealth(ctx context.Context, serviceName string) ([]*domain.SystemHealth, error) {
	if q.healthReadRepo != nil {
		if cached, err := q.healthReadRepo.ListLatest(ctx, serviceName, 10); err == nil && len(cached) > 0 {
			return cached, nil
		}
	}
	return q.healthRepo.GetLatestHealth(ctx, serviceName, 10)
}

func (q *MonitoringAnalyticsQueryService) GetAlerts(ctx context.Context, limit int) ([]*domain.Alert, error) {
	if q.alertReadRepo != nil {
		if cached, err := q.alertReadRepo.ListLatest(ctx, limit); err == nil && len(cached) > 0 {
			return cached, nil
		}
	}
	return q.alertRepo.GetAlerts(ctx, limit)
}
