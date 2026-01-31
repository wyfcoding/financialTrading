package domain

// EventPublisher 事件发布者接口
type EventPublisher interface {
	// PublishNotificationCreated 发布通知创建事件
	PublishNotificationCreated(event NotificationCreatedEvent) error

	// PublishNotificationSent 发布通知发送成功事件
	PublishNotificationSent(event NotificationSentEvent) error

	// PublishNotificationFailed 发布通知发送失败事件
	PublishNotificationFailed(event NotificationFailedEvent) error

	// PublishNotificationStatusChanged 发布通知状态变更事件
	PublishNotificationStatusChanged(event NotificationStatusChangedEvent) error

	// PublishBatchNotificationCreated 发布批量通知创建事件
	PublishBatchNotificationCreated(event BatchNotificationCreatedEvent) error
}
