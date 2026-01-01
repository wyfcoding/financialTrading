// Package domain 通知服务的领域模型
package domain

import (
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

// End of domain file
