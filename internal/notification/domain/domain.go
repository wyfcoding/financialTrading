// Package domain 通知服务的领域模型
package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// NotificationType 通知类型
type NotificationType string

const (
	NotificationTypeEmail   NotificationType = "EMAIL"   // 邮件通知
	NotificationTypeSMS     NotificationType = "SMS"     // 短信通知
	NotificationTypeWebhook NotificationType = "WEBHOOK" // Webhook 通知
)

// NotificationStatus 通知状态
type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "PENDING"
	NotificationStatusSent    NotificationStatus = "SENT"
	NotificationStatusFailed  NotificationStatus = "FAILED"
)

// Notification 通知实体
type Notification struct {
	gorm.Model
	// NotificationID 通知 ID
	NotificationID string `gorm:"column:notification_id;type:varchar(32);uniqueIndex;not null" json:"notification_id"`
	// UserID 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(32);index;not null" json:"user_id"`
	// Type 通知类型
	Type NotificationType `gorm:"column:type;type:varchar(20);not null" json:"type"`
	// Subject 通知主题
	Subject string `gorm:"column:subject;type:varchar(100)" json:"subject"`
	// Content 通知内容
	Content string `gorm:"column:content;type:text" json:"content"`
	// Target 通知目标（如邮箱、手机号）
	Target string `gorm:"column:target;type:varchar(100);not null" json:"target"`
	// Status 通知状态
	Status NotificationStatus `gorm:"column:status;type:varchar(20);index;not null;default:'PENDING'" json:"status"`
	// ErrorMessage 错误信息
	ErrorMessage string `gorm:"column:error_message;type:text" json:"error_message"`
	// SentAt 发送时间
	SentAt *time.Time `gorm:"column:sent_at;type:datetime" json:"sent_at"`
}

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
