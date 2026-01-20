package domain

import (
	"gorm.io/gorm"
)

type NotificationStatus string

const (
	StatusPending NotificationStatus = "pending"
	StatusSent    NotificationStatus = "sent"
	StatusFailed  NotificationStatus = "failed"
)

type Channel string

const (
	ChannelEmail Channel = "email"
	ChannelSMS   Channel = "sms"
	ChannelPush  Channel = "push"
)

// Notification represents a message sent to a user
type Notification struct {
	gorm.Model
	NotificationID string             `gorm:"column:notification_id;type:varchar(50);uniqueIndex" json:"notification_id"`
	UserID         string             `gorm:"column:user_id;type:varchar(50);index" json:"user_id"`
	Channel        Channel            `gorm:"column:channel;type:varchar(20)" json:"channel"`
	Recipient      string             `gorm:"column:recipient;type:varchar(100)" json:"recipient"`
	Subject        string             `gorm:"column:subject;type:varchar(255)" json:"subject"`
	Content        string             `gorm:"column:content;type:text" json:"content"`
	Status         NotificationStatus `gorm:"column:status;type:varchar(20);index" json:"status"`
	ErrorMsg       string             `gorm:"column:error_msg;type:text" json:"error_msg"`
}

// TableName overrides the table name
func (Notification) TableName() string {
	return "notifications"
}

// NewNotification creates a new notification
func NewNotification(userID string, channel Channel, recipient, subject, content string) *Notification {
	return &Notification{
		UserID:    userID,
		Channel:   channel,
		Recipient: recipient,
		Subject:   subject,
		Content:   content,
		Status:    StatusPending,
	}
}

func (n *Notification) MarkSent() {
	n.Status = StatusSent
}

func (n *Notification) MarkFailed(err string) {
	n.Status = StatusFailed
	n.ErrorMsg = err
}
