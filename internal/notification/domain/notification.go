package domain

import (
	"errors"
	"slices"
	"time"
)

type NotificationType string

const (
	NotificationTypeTrade      NotificationType = "TRADE"
	NotificationTypeOrder      NotificationType = "ORDER"
	NotificationTypePosition   NotificationType = "POSITION"
	NotificationTypeRisk       NotificationType = "RISK"
	NotificationTypeMargin     NotificationType = "MARGIN"
	NotificationTypeSettlement NotificationType = "SETTLEMENT"
	NotificationTypeDeposit    NotificationType = "DEPOSIT"
	NotificationTypeWithdrawal NotificationType = "WITHDRAWAL"
	NotificationTypeKYC        NotificationType = "KYC"
	NotificationTypeAML        NotificationType = "AML"
	NotificationTypeSystem     NotificationType = "SYSTEM"
	NotificationTypePriceAlert NotificationType = "PRICE_ALERT"
	NotificationTypeExecution  NotificationType = "EXECUTION"
)

type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "LOW"
	PriorityNormal   NotificationPriority = "NORMAL"
	PriorityHigh     NotificationPriority = "HIGH"
	PriorityCritical NotificationPriority = "CRITICAL"
)

type NotificationStatus string

const (
	StatusPending   NotificationStatus = "PENDING"
	StatusSent      NotificationStatus = "SENT"
	StatusDelivered NotificationStatus = "DELIVERED"
	StatusFailed    NotificationStatus = "FAILED"
	StatusRead      NotificationStatus = "READ"
)

type NotificationChannel string

const (
	ChannelEmail     NotificationChannel = "EMAIL"
	ChannelSMS       NotificationChannel = "SMS"
	ChannelPush      NotificationChannel = "PUSH"
	ChannelInApp     NotificationChannel = "IN_APP"
	ChannelWebhook   NotificationChannel = "WEBHOOK"
	ChannelWebSocket NotificationChannel = "WEBSOCKET"
)

type Notification struct {
	ID             uint64                `json:"id"`
	NotificationID string                `json:"notification_id"`
	UserID         uint64                `json:"user_id"`
	Type           NotificationType      `json:"type"`
	Priority       NotificationPriority  `json:"priority"`
	Title          string                `json:"title"`
	Content        string                `json:"content"`
	Data           map[string]any        `json:"data"`
	Channels       []NotificationChannel `json:"channels"`
	Status         NotificationStatus    `json:"status"`
	SentAt         *time.Time            `json:"sent_at"`
	DeliveredAt    *time.Time            `json:"delivered_at"`
	ReadAt         *time.Time            `json:"read_at"`
	FailReason     string                `json:"fail_reason"`
	RetryCount     int                   `json:"retry_count"`
	MaxRetries     int                   `json:"max_retries"`
	ScheduledAt    *time.Time            `json:"scheduled_at"`
	ExpiresAt      *time.Time            `json:"expires_at"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

func NewNotification(notificationID string, userID uint64, notifType NotificationType, priority NotificationPriority, title, content string) *Notification {
	return &Notification{
		NotificationID: notificationID,
		UserID:         userID,
		Type:           notifType,
		Priority:       priority,
		Title:          title,
		Content:        content,
		Data:           make(map[string]any),
		Channels:       make([]NotificationChannel, 0),
		Status:         StatusPending,
		RetryCount:     0,
		MaxRetries:     3,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func (n *Notification) SetData(data map[string]any) {
	n.Data = data
	n.UpdatedAt = time.Now()
}

func (n *Notification) AddChannel(channel NotificationChannel) {
	if slices.Contains(n.Channels, channel) {
		return
	}
	n.Channels = append(n.Channels, channel)
	n.UpdatedAt = time.Now()
}

func (n *Notification) SetSchedule(scheduledAt, expiresAt *time.Time) {
	n.ScheduledAt = scheduledAt
	n.ExpiresAt = expiresAt
	n.UpdatedAt = time.Now()
}

func (n *Notification) MarkSent() {
	now := time.Now()
	n.Status = StatusSent
	n.SentAt = &now
	n.UpdatedAt = now
}

func (n *Notification) MarkDelivered() {
	now := time.Now()
	n.Status = StatusDelivered
	n.DeliveredAt = &now
	n.UpdatedAt = now
}

func (n *Notification) MarkFailed(reason string) {
	n.Status = StatusFailed
	n.FailReason = reason
	n.RetryCount++
	n.UpdatedAt = time.Now()
}

func (n *Notification) MarkRead() {
	now := time.Now()
	n.Status = StatusRead
	n.ReadAt = &now
	n.UpdatedAt = now
}

func (n *Notification) CanRetry() bool {
	return n.RetryCount < n.MaxRetries
}

func (n *Notification) IsExpired() bool {
	if n.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*n.ExpiresAt)
}

type NotificationTemplate struct {
	ID        uint64                `json:"id"`
	Code      string                `json:"code"`
	Type      NotificationType      `json:"type"`
	Title     string                `json:"title"`
	Content   string                `json:"content"`
	Channels  []NotificationChannel `json:"channels"`
	IsActive  bool                  `json:"is_active"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

func NewNotificationTemplate(code string, notifType NotificationType, title, content string) *NotificationTemplate {
	return &NotificationTemplate{
		Code:      code,
		Type:      notifType,
		Title:     title,
		Content:   content,
		Channels:  make([]NotificationChannel, 0),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

type UserNotificationPreference struct {
	ID         uint64                `json:"id"`
	UserID     uint64                `json:"user_id"`
	Type       NotificationType      `json:"type"`
	Channels   []NotificationChannel `json:"channels"`
	Enabled    bool                  `json:"enabled"`
	QuietHours *QuietHours           `json:"quiet_hours"`
	CreatedAt  time.Time             `json:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
}

type QuietHours struct {
	StartHour int `json:"start_hour"`
	StartMin  int `json:"start_min"`
	EndHour   int `json:"end_hour"`
	EndMin    int `json:"end_min"`
}

func (q *QuietHours) IsQuietTime(t time.Time) bool {
	hour, min := t.Hour(), t.Minute()
	startMins := q.StartHour*60 + q.StartMin
	endMins := q.EndHour*60 + q.EndMin
	currentMins := hour*60 + min

	if startMins < endMins {
		return currentMins >= startMins && currentMins < endMins
	}
	return currentMins >= startMins || currentMins < endMins
}

func NewUserNotificationPreference(userID uint64, notifType NotificationType) *UserNotificationPreference {
	return &UserNotificationPreference{
		UserID:    userID,
		Type:      notifType,
		Channels:  []NotificationChannel{ChannelInApp},
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (p *UserNotificationPreference) SetChannels(channels []NotificationChannel) {
	p.Channels = channels
	p.UpdatedAt = time.Now()
}

func (p *UserNotificationPreference) SetEnabled(enabled bool) {
	p.Enabled = enabled
	p.UpdatedAt = time.Now()
}

func (p *UserNotificationPreference) SetQuietHours(startHour, startMin, endHour, endMin int) {
	p.QuietHours = &QuietHours{
		StartHour: startHour,
		StartMin:  startMin,
		EndHour:   endHour,
		EndMin:    endMin,
	}
	p.UpdatedAt = time.Now()
}

type NotificationBatch struct {
	ID          uint64           `json:"id"`
	BatchID     string           `json:"batch_id"`
	Type        NotificationType `json:"type"`
	Title       string           `json:"title"`
	Content     string           `json:"content"`
	UserIDs     []uint64         `json:"user_ids"`
	TotalCount  int              `json:"total_count"`
	SentCount   int              `json:"sent_count"`
	FailCount   int              `json:"fail_count"`
	Status      string           `json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	CompletedAt *time.Time       `json:"completed_at"`
}

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrTemplateNotFound     = errors.New("template not found")
	ErrPreferenceNotFound   = errors.New("preference not found")
	ErrInvalidChannel       = errors.New("invalid notification channel")
	ErrMaxRetriesExceeded   = errors.New("max retries exceeded")
)
