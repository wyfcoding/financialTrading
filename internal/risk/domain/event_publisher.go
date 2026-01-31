package domain

// EventPublisher 事件发布者接口
type EventPublisher interface {
	// PublishRiskAssessmentCreated 发布风险评估创建事件
	PublishRiskAssessmentCreated(event RiskAssessmentCreatedEvent) error

	// PublishRiskLimitExceeded 发布风险限额超出事件
	PublishRiskLimitExceeded(event RiskLimitExceededEvent) error

	// PublishCircuitBreakerFired 发布熔断触发事件
	PublishCircuitBreakerFired(event CircuitBreakerFiredEvent) error

	// PublishCircuitBreakerReset 发布熔断重置事件
	PublishCircuitBreakerReset(event CircuitBreakerResetEvent) error

	// PublishRiskAlertGenerated 发布风险告警生成事件
	PublishRiskAlertGenerated(event RiskAlertGeneratedEvent) error

	// PublishMarginCall 发布追加保证金通知事件
	PublishMarginCall(event MarginCallEvent) error

	// PublishRiskMetricsUpdated 发布风险指标更新事件
	PublishRiskMetricsUpdated(event RiskMetricsUpdatedEvent) error

	// PublishRiskLevelChanged 发布风险等级变更事件
	PublishRiskLevelChanged(event RiskLevelChangedEvent) error
}
