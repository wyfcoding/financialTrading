package domain

import (
	"context"
)

// NotificationRepository 通知仓储接口
type NotificationRepository interface {
	// Save 保存或更新通知记录
	Save(ctx context.Context, notification *Notification) error
	// Get 根据通知 ID 获取通知记录
	Get(ctx context.Context, notificationID string) (*Notification, error)
	// ListByUserID 分页获取指定用户的通知列表
	ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*Notification, int64, error)
}

// Sender 通知发送接口
type Sender interface {
	Send(ctx context.Context, target string, subject string, content string) error
}

// EmailSender 邮件发送接口
type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, content string) error
}

// SMSSender 短信发送接口
type SMSSender interface {
	SendSMS(ctx context.Context, to, content string) error
}
