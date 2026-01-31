package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
)

// MonitoringAnalyticsCommand 处理监控分析相关的命令操作
type MonitoringAnalyticsCommand struct {
	metricRepo     domain.MetricRepository
	healthRepo     domain.SystemHealthRepository
	alertRepo      domain.AlertRepository
	auditRepo      domain.ExecutionAuditRepository
	auditESRepo    domain.AuditESRepository
	eventPublisher domain.EventPublisher
}

// NewMonitoringAnalyticsCommand 创建新的 MonitoringAnalyticsCommand 实例
func NewMonitoringAnalyticsCommand(
	metricRepo domain.MetricRepository,
	healthRepo domain.SystemHealthRepository,
	alertRepo domain.AlertRepository,
	auditRepo domain.ExecutionAuditRepository,
	auditESRepo domain.AuditESRepository,
	eventPublisher domain.EventPublisher,
) *MonitoringAnalyticsCommand {
	return &MonitoringAnalyticsCommand{
		metricRepo:     metricRepo,
		healthRepo:     healthRepo,
		alertRepo:      alertRepo,
		auditRepo:      auditRepo,
		auditESRepo:    auditESRepo,
		eventPublisher: eventPublisher,
	}
}

// RecordMetric 记录指标
func (c *MonitoringAnalyticsCommand) RecordMetric(ctx context.Context, name string, value decimal.Decimal, tags map[string]string, timestamp int64) error {
	// 创建并保存指标
	metric := &domain.Metric{
		Name:      name,
		Value:     value,
		Tags:      tags,
		Timestamp: timestamp,
	}

	if err := c.metricRepo.Save(ctx, metric); err != nil {
		return err
	}

	// 发布指标创建事件
	event := domain.MetricCreatedEvent{
		MetricName: name,
		Value:      value,
		Tags:       tags,
		Timestamp:  timestamp,
		OccurredOn: time.Now(),
	}

	return c.eventPublisher.PublishMetricCreated(event)
}

// SaveSystemHealth 保存系统健康状态
func (c *MonitoringAnalyticsCommand) SaveSystemHealth(ctx context.Context, health *domain.SystemHealth) error {
	// 保存系统健康状态
	if err := c.healthRepo.Save(ctx, health); err != nil {
		return err
	}

	// 发布系统健康状态变更事件
	event := domain.SystemHealthChangedEvent{
		ServiceName: health.ServiceName,
		OldStatus:   "", // 实际应用中需要获取旧状态
		NewStatus:   health.Status,
		CPUUsage:    health.CPUUsage,
		MemoryUsage: health.MemoryUsage,
		Message:     health.Message,
		LastChecked: health.LastChecked,
		OccurredOn:  time.Now(),
	}

	return c.eventPublisher.PublishSystemHealthChanged(event)
}

// CreateAlert 创建告警
func (c *MonitoringAnalyticsCommand) CreateAlert(ctx context.Context, alert *domain.Alert) error {
	// 保存告警
	if err := c.alertRepo.Save(ctx, alert); err != nil {
		return err
	}

	// 发布告警生成事件
	event := domain.AlertGeneratedEvent{
		AlertID:     alert.AlertID,
		RuleName:    alert.RuleName,
		Severity:    alert.Severity,
		Message:     alert.Message,
		Source:      alert.Source,
		GeneratedAt: alert.GeneratedAt,
		OccurredOn:  time.Now(),
	}

	return c.eventPublisher.PublishAlertGenerated(event)
}

// UpdateAlertStatus 更新告警状态
func (c *MonitoringAnalyticsCommand) UpdateAlertStatus(ctx context.Context, alertID string, oldStatus, newStatus string) error {
	// 发布告警状态变更事件
	event := domain.AlertStatusChangedEvent{
		AlertID:    alertID,
		OldStatus:  oldStatus,
		NewStatus:  newStatus,
		UpdatedAt:  time.Now().Unix(),
		OccurredOn: time.Now(),
	}

	return c.eventPublisher.PublishAlertStatusChanged(event)
}

// RecordExecutionAudit 记录执行审计
func (c *MonitoringAnalyticsCommand) RecordExecutionAudit(ctx context.Context, audit *domain.ExecutionAudit) error {
	// 1. 持久化到 ClickHouse (批量保存建议由上层或异步处理，这里为了简化直接保存)
	_ = c.auditRepo.BatchSave(ctx, []*domain.ExecutionAudit{audit})

	// 2. 索引到 Elasticsearch
	_ = c.auditESRepo.Index(ctx, audit)

	// 3. 发布执行审计创建事件
	event := domain.ExecutionAuditCreatedEvent{
		ID:         audit.ID,
		TradeID:    audit.TradeID,
		OrderID:    audit.OrderID,
		UserID:     audit.UserID,
		Symbol:     audit.Symbol,
		Side:       audit.Side,
		Price:      audit.Price,
		Quantity:   audit.Quantity,
		Fee:        audit.Fee,
		Venue:      audit.Venue,
		AlgoType:   audit.AlgoType,
		Timestamp:  audit.Timestamp,
		OccurredOn: time.Now(),
	}

	return c.eventPublisher.PublishExecutionAuditCreated(event)
}

// RecordSpoofingDetection 记录哄骗检测
func (c *MonitoringAnalyticsCommand) RecordSpoofingDetection(ctx context.Context, userID, symbol, orderID string) error {
	// 发布哄骗检测事件
	event := domain.SpoofingDetectedEvent{
		UserID:     userID,
		Symbol:     symbol,
		OrderID:    orderID,
		DetectedAt: time.Now().Unix(),
		OccurredOn: time.Now(),
	}

	return c.eventPublisher.PublishSpoofingDetected(event)
}

// RecordMarketAnomaly 记录市场异常
func (c *MonitoringAnalyticsCommand) RecordMarketAnomaly(ctx context.Context, symbol, anomalyType string, details map[string]interface{}) error {
	// 发布市场异常检测事件
	event := domain.MarketAnomalyDetectedEvent{
		Symbol:      symbol,
		AnomalyType: anomalyType,
		Details:     details,
		DetectedAt:  time.Now().Unix(),
		OccurredOn:  time.Now(),
	}

	return c.eventPublisher.PublishMarketAnomalyDetected(event)
}
