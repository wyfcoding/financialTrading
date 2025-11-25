package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// NotificationType 通知类型
type NotificationType string

const (
	NotificationTypeEmail   NotificationType = "EMAIL"
	NotificationTypeSMS     NotificationType = "SMS"
	NotificationTypeWebhook NotificationType = "WEBHOOK"
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
	ID           string             `gorm:"column:id;type:varchar(36);primaryKey" json:"id"`
	UserID       string             `gorm:"column:user_id;type:varchar(36);index;not null" json:"user_id"`
	Type         NotificationType   `gorm:"column:type;type:varchar(20);not null" json:"type"`
	Subject      string             `gorm:"column:subject;type:varchar(255)" json:"subject"`
	Content      string             `gorm:"column:content;type:text" json:"content"`
	Target       string             `gorm:"column:target;type:varchar(255);not null" json:"target"`
	Status       NotificationStatus `gorm:"column:status;type:varchar(20);default:'PENDING';index" json:"status"`
	ErrorMessage string             `gorm:"column:error_message;type:text" json:"error_message"`
	CreatedAt    time.Time          `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
	SentAt       time.Time          `gorm:"column:sent_at;type:datetime" json:"sent_at"`
}

// NotificationRepository 通知仓储接口
type NotificationRepository interface {
	Save(ctx context.Context, notification *Notification) error
	GetByID(ctx context.Context, id string) (*Notification, error)
	ListByUserID(ctx context.Context, userID string, limit int, offset int) ([]*Notification, error)
}

// Sender 通知发送接口
type Sender interface {
	Send(ctx context.Context, target string, subject string, content string) error
}
