package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/messagequeue"
)

// MonitoringAnalyticsCommandService 处理监控分析相关的命令操作
type MonitoringAnalyticsCommandService struct {
	metricRepo     domain.MetricRepository
	metricReadRepo domain.MetricReadRepository
	healthRepo     domain.SystemHealthRepository
	healthReadRepo domain.SystemHealthReadRepository
	alertRepo      domain.AlertRepository
	alertReadRepo  domain.AlertReadRepository
	auditRepo      domain.ExecutionAuditRepository
	auditESRepo    domain.AuditESRepository
	eventPublisher messagequeue.EventPublisher
}

// NewMonitoringAnalyticsCommandService 创建新的命令服务实例
func NewMonitoringAnalyticsCommandService(
	metricRepo domain.MetricRepository,
	metricReadRepo domain.MetricReadRepository,
	healthRepo domain.SystemHealthRepository,
	healthReadRepo domain.SystemHealthReadRepository,
	alertRepo domain.AlertRepository,
	alertReadRepo domain.AlertReadRepository,
	auditRepo domain.ExecutionAuditRepository,
	auditESRepo domain.AuditESRepository,
	eventPublisher messagequeue.EventPublisher,
) *MonitoringAnalyticsCommandService {
	return &MonitoringAnalyticsCommandService{
		metricRepo:     metricRepo,
		metricReadRepo: metricReadRepo,
		healthRepo:     healthRepo,
		healthReadRepo: healthReadRepo,
		alertRepo:      alertRepo,
		alertReadRepo:  alertReadRepo,
		auditRepo:      auditRepo,
		auditESRepo:    auditESRepo,
		eventPublisher: eventPublisher,
	}
}

// RecordMetric 记录指标
func (c *MonitoringAnalyticsCommandService) RecordMetric(ctx context.Context, name string, value decimal.Decimal, tags map[string]string, timestamp int64) error {
	metric := &domain.Metric{
		Name:      name,
		Value:     value,
		Tags:      tags,
		Timestamp: timestamp,
	}

	err := c.metricRepo.WithTx(ctx, func(txCtx context.Context) error {
		if err := c.metricRepo.Save(txCtx, metric); err != nil {
			return err
		}
		if c.eventPublisher != nil {
			event := domain.MetricCreatedEvent{
				MetricName: name,
				Value:      value,
				Tags:       tags,
				Timestamp:  timestamp,
				OccurredOn: time.Now(),
			}
			if err := c.eventPublisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.MetricCreatedEventType, name, event); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if c.metricReadRepo != nil {
		_ = c.metricReadRepo.Save(ctx, metric)
	}
	return nil
}

// SaveSystemHealth 保存系统健康状态
func (c *MonitoringAnalyticsCommandService) SaveSystemHealth(ctx context.Context, health *domain.SystemHealth) error {
	// 保存系统健康状态
	err := c.healthRepo.WithTx(ctx, func(txCtx context.Context) error {
		if err := c.healthRepo.Save(txCtx, health); err != nil {
			return err
		}
		if c.eventPublisher != nil {
			event := domain.SystemHealthChangedEvent{
				ServiceName: health.ServiceName,
				OldStatus:   "",
				NewStatus:   health.Status,
				CPUUsage:    health.CPUUsage,
				MemoryUsage: health.MemoryUsage,
				Message:     health.Message,
				LastChecked: health.LastChecked,
				OccurredOn:  time.Now(),
			}
			if err := c.eventPublisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.SystemHealthChangedEventType, health.ServiceName, event); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if c.healthReadRepo != nil {
		_ = c.healthReadRepo.Save(ctx, health)
	}
	return nil
}

// CreateAlert 创建告警
func (c *MonitoringAnalyticsCommandService) CreateAlert(ctx context.Context, alert *domain.Alert) error {
	// 保存告警
	err := c.alertRepo.WithTx(ctx, func(txCtx context.Context) error {
		if err := c.alertRepo.Save(txCtx, alert); err != nil {
			return err
		}
		if c.eventPublisher != nil {
			event := domain.AlertGeneratedEvent{
				AlertID:     alert.AlertID,
				RuleName:    alert.RuleName,
				Severity:    alert.Severity,
				Message:     alert.Message,
				Source:      alert.Source,
				GeneratedAt: alert.GeneratedAt,
				OccurredOn:  time.Now(),
			}
			if err := c.eventPublisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.AlertGeneratedEventType, alert.AlertID, event); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if c.alertReadRepo != nil {
		_ = c.alertReadRepo.Save(ctx, alert)
	}
	return nil
}

// UpdateAlertStatus 更新告警状态
func (c *MonitoringAnalyticsCommandService) UpdateAlertStatus(ctx context.Context, alertID string, oldStatus, newStatus string) error {
	if err := c.alertRepo.UpdateStatus(ctx, alertID, newStatus); err != nil {
		return err
	}
	if c.eventPublisher != nil {
		event := domain.AlertStatusChangedEvent{
			AlertID:    alertID,
			OldStatus:  oldStatus,
			NewStatus:  newStatus,
			UpdatedAt:  time.Now().Unix(),
			OccurredOn: time.Now(),
		}
		return c.eventPublisher.Publish(ctx, domain.AlertStatusChangedEventType, alertID, event)
	}
	return nil
}

// RecordExecutionAudit 记录执行审计
func (c *MonitoringAnalyticsCommandService) RecordExecutionAudit(ctx context.Context, audit *domain.ExecutionAudit) error {
	// 1. 持久化到 ClickHouse (批量保存建议由上层或异步处理，这里为了简化直接保存)
	if c.auditRepo != nil {
		_ = c.auditRepo.BatchSave(ctx, []*domain.ExecutionAudit{audit})
	}

	// 2. 索引到 Elasticsearch
	if c.auditESRepo != nil {
		_ = c.auditESRepo.Index(ctx, audit)
	}

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

	if c.eventPublisher != nil {
		return c.eventPublisher.Publish(ctx, domain.ExecutionAuditCreatedEventType, audit.ID, event)
	}
	return nil
}

// RecordSpoofingDetection 记录哄骗检测
func (c *MonitoringAnalyticsCommandService) RecordSpoofingDetection(ctx context.Context, userID, symbol, orderID string) error {
	// 发布哄骗检测事件
	event := domain.SpoofingDetectedEvent{
		UserID:     userID,
		Symbol:     symbol,
		OrderID:    orderID,
		DetectedAt: time.Now().Unix(),
		OccurredOn: time.Now(),
	}

	if c.eventPublisher != nil {
		return c.eventPublisher.Publish(ctx, domain.SpoofingDetectedEventType, userID, event)
	}
	return nil
}

// RecordMarketAnomaly 记录市场异常
func (c *MonitoringAnalyticsCommandService) RecordMarketAnomaly(ctx context.Context, symbol, anomalyType string, details map[string]any) error {
	// 发布市场异常检测事件
	event := domain.MarketAnomalyDetectedEvent{
		Symbol:      symbol,
		AnomalyType: anomalyType,
		Details:     details,
		DetectedAt:  time.Now().Unix(),
		OccurredOn:  time.Now(),
	}

	if c.eventPublisher != nil {
		return c.eventPublisher.Publish(ctx, domain.MarketAnomalyDetectedEventType, symbol, event)
	}
	return nil
}
