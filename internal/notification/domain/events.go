package domain

import (
	"time"
)

// NotificationCreatedEvent 通知创建事件
type NotificationCreatedEvent struct {
	NotificationID string
	UserID         string
	Channel        Channel
	Recipient      string
	Subject        string
	Content        string
	Status         NotificationStatus
	OccurredOn     time.Time
}

// NotificationSentEvent 通知发送成功事件
type NotificationSentEvent struct {
	NotificationID string
	UserID         string
	Channel        Channel
	Recipient      string
	SentAt         int64
	OccurredOn     time.Time
}

// NotificationFailedEvent 通知发送失败事件
type NotificationFailedEvent struct {
	NotificationID string
	UserID         string
	Channel        Channel
	Recipient      string
	ErrorMsg       string
	FailedAt       int64
	OccurredOn     time.Time
}

// NotificationStatusChangedEvent 通知状态变更事件
type NotificationStatusChangedEvent struct {
	NotificationID string
	OldStatus      NotificationStatus
	NewStatus      NotificationStatus
	UpdatedAt      int64
	OccurredOn     time.Time
}

// BatchNotificationCreatedEvent 批量通知创建事件
type BatchNotificationCreatedEvent struct {
	BatchID         string
	NotificationIDs []string
	UserIDs         []string
	Channel         Channel
	Count           int
	CreatedAt       int64
	OccurredOn      time.Time
}
