package domain

// EventPublisher 事件发布者接口
type EventPublisher interface {
	// PublishPositionCreated 发布头寸创建事件
	PublishPositionCreated(event PositionCreatedEvent) error

	// PublishPositionUpdated 发布头寸更新事件
	PublishPositionUpdated(event PositionUpdatedEvent) error

	// PublishPositionClosed 发布头寸关闭事件
	PublishPositionClosed(event PositionClosedEvent) error

	// PublishPositionPnLUpdated 发布头寸盈亏更新事件
	PublishPositionPnLUpdated(event PositionPnLUpdatedEvent) error

	// PublishPositionCostMethodChanged 发布头寸成本计算方法变更事件
	PublishPositionCostMethodChanged(event PositionCostMethodChangedEvent) error

	// PublishPositionFlip 发布头寸反手事件
	PublishPositionFlip(event PositionFlipEvent) error
}
