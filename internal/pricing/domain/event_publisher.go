package domain

// EventPublisher 事件发布者接口
type EventPublisher interface {
	// PublishOptionPriced 发布期权定价完成事件
	PublishOptionPriced(event OptionPricedEvent) error

	// PublishGreeksCalculated 发布希腊字母计算完成事件
	PublishGreeksCalculated(event GreeksCalculatedEvent) error

	// PublishPricingModelChanged 发布定价模型变更事件
	PublishPricingModelChanged(event PricingModelChangedEvent) error

	// PublishVolatilityUpdated 发布波动率更新事件
	PublishVolatilityUpdated(event VolatilityUpdatedEvent) error

	// PublishPricingError 发布定价错误事件
	PublishPricingError(event PricingErrorEvent) error

	// PublishBatchPricingCompleted 发布批量定价完成事件
	PublishBatchPricingCompleted(event BatchPricingCompletedEvent) error
}
