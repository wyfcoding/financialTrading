package domain

// EventPublisher 事件发布者接口
type EventPublisher interface {
	// PublishOrderCreated 发布订单创建事件
	PublishOrderCreated(event OrderCreatedEvent) error

	// PublishOrderValidated 发布订单验证通过事件
	PublishOrderValidated(event OrderValidatedEvent) error

	// PublishOrderRejected 发布订单被拒绝事件
	PublishOrderRejected(event OrderRejectedEvent) error

	// PublishOrderPartiallyFilled 发布订单部分成交事件
	PublishOrderPartiallyFilled(event OrderPartiallyFilledEvent) error

	// PublishOrderFilled 发布订单完全成交事件
	PublishOrderFilled(event OrderFilledEvent) error

	// PublishOrderCancelled 发布订单被取消事件
	PublishOrderCancelled(event OrderCancelledEvent) error

	// PublishOrderExpired 发布订单过期事件
	PublishOrderExpired(event OrderExpiredEvent) error

	// PublishOrderStatusChanged 发布订单状态变更事件
	PublishOrderStatusChanged(event OrderStatusChangedEvent) error
}
