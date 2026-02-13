package domain

import (
	"context"
	"time"
)

type NotificationRepository interface {
	Save(ctx context.Context, notification *Notification) error
	Update(ctx context.Context, notification *Notification) error
	GetByID(ctx context.Context, id uint64) (*Notification, error)
	GetByNotificationID(ctx context.Context, notificationID string) (*Notification, error)
	ListByUserID(ctx context.Context, userID uint64, status NotificationStatus, page, pageSize int) ([]*Notification, int64, error)
	ListPending(ctx context.Context, limit int) ([]*Notification, error)
	ListScheduled(ctx context.Context, before time.Time, limit int) ([]*Notification, error)
	MarkBatchSent(ctx context.Context, ids []uint64) error
	MarkBatchFailed(ctx context.Context, ids []uint64, reason string) error
}

type NotificationTemplateRepository interface {
	Save(ctx context.Context, template *NotificationTemplate) error
	GetByCode(ctx context.Context, code string) (*NotificationTemplate, error)
	GetByType(ctx context.Context, notifType NotificationType) ([]*NotificationTemplate, error)
	ListActive(ctx context.Context) ([]*NotificationTemplate, error)
}

type UserNotificationPreferenceRepository interface {
	Save(ctx context.Context, pref *UserNotificationPreference) error
	GetByUserIDAndType(ctx context.Context, userID uint64, notifType NotificationType) (*UserNotificationPreference, error)
	GetByUserID(ctx context.Context, userID uint64) ([]*UserNotificationPreference, error)
}

type NotificationBatchRepository interface {
	Save(ctx context.Context, batch *NotificationBatch) error
	Update(ctx context.Context, batch *NotificationBatch) error
	GetByID(ctx context.Context, id uint64) (*NotificationBatch, error)
	GetByBatchID(ctx context.Context, batchID string) (*NotificationBatch, error)
}

type NotificationSender interface {
	Send(ctx context.Context, notification *Notification, channel NotificationChannel) error
	SendBatch(ctx context.Context, notifications []*Notification, channel NotificationChannel) error
}

type EmailSender interface {
	Send(ctx context.Context, to, subject, content string, data map[string]any) error
}

type SMSSender interface {
	Send(ctx context.Context, to, content string) error
}

type PushSender interface {
	Send(ctx context.Context, userID uint64, title, content string, data map[string]any) error
}

type WebSocketSender interface {
	Broadcast(ctx context.Context, userID uint64, message any) error
}

type WebhookSender interface {
	Send(ctx context.Context, url string, payload any) error
}
