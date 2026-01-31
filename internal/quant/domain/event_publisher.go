package domain

// EventPublisher 事件发布者接口
type EventPublisher interface {
	// PublishStrategyCreated 发布策略创建事件
	PublishStrategyCreated(event StrategyCreatedEvent) error

	// PublishStrategyUpdated 发布策略更新事件
	PublishStrategyUpdated(event StrategyUpdatedEvent) error

	// PublishStrategyDeleted 发布策略删除事件
	PublishStrategyDeleted(event StrategyDeletedEvent) error

	// PublishBacktestStarted 发布回测开始事件
	PublishBacktestStarted(event BacktestStartedEvent) error

	// PublishBacktestCompleted 发布回测完成事件
	PublishBacktestCompleted(event BacktestCompletedEvent) error

	// PublishBacktestFailed 发布回测失败事件
	PublishBacktestFailed(event BacktestFailedEvent) error

	// PublishSignalGenerated 发布信号生成事件
	PublishSignalGenerated(event SignalGeneratedEvent) error

	// PublishPortfolioOptimized 发布组合优化事件
	PublishPortfolioOptimized(event PortfolioOptimizedEvent) error

	// PublishRiskAssessmentCompleted 发布风险评估完成事件
	PublishRiskAssessmentCompleted(event RiskAssessmentCompletedEvent) error
}
