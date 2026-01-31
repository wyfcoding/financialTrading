package domain

// EventPublisher 事件发布者接口
type EventPublisher interface {
	// PublishMetricCreated 发布指标创建事件
	PublishMetricCreated(event MetricCreatedEvent) error

	// PublishAlertGenerated 发布告警生成事件
	PublishAlertGenerated(event AlertGeneratedEvent) error

	// PublishAlertStatusChanged 发布告警状态变更事件
	PublishAlertStatusChanged(event AlertStatusChangedEvent) error

	// PublishSystemHealthChanged 发布系统健康状态变更事件
	PublishSystemHealthChanged(event SystemHealthChangedEvent) error

	// PublishExecutionAuditCreated 发布执行审计创建事件
	PublishExecutionAuditCreated(event ExecutionAuditCreatedEvent) error

	// PublishSpoofingDetected 发布哄骗检测事件
	PublishSpoofingDetected(event SpoofingDetectedEvent) error

	// PublishMarketAnomalyDetected 发布市场异常检测事件
	PublishMarketAnomalyDetected(event MarketAnomalyDetectedEvent) error
}
