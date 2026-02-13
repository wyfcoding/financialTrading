package infrastructure

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
	"gorm.io/gorm"
)

type NotificationPO struct {
	ID             uint64   `gorm:"column:id;primaryKey;autoIncrement"`
	NotificationID string   `gorm:"column:notification_id;type:varchar(32);uniqueIndex;not null"`
	UserID         uint64   `gorm:"column:user_id;index;not null"`
	Type           string   `gorm:"column:type;type:varchar(20);not null"`
	Priority       string   `gorm:"column:priority;type:varchar(20);not null"`
	Title          string   `gorm:"column:title;type:varchar(255);not null"`
	Content        string   `gorm:"column:content;type:text"`
	Data           string   `gorm:"column:data;type:json"`
	Channels       string   `gorm:"column:channels;type:json"`
	Status         string   `gorm:"column:status;type:varchar(20);not null;default:'PENDING'"`
	SentAt         *time.Time `gorm:"column:sent_at"`
	DeliveredAt    *time.Time `gorm:"column:delivered_at"`
	ReadAt         *time.Time `gorm:"column:read_at"`
	FailReason     string   `gorm:"column:fail_reason;type:varchar(255)"`
	RetryCount     int      `gorm:"column:retry_count;default:0"`
	MaxRetries     int      `gorm:"column:max_retries;default:3"`
	ScheduledAt    *time.Time `gorm:"column:scheduled_at"`
	ExpiresAt      *time.Time `gorm:"column:expires_at"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (NotificationPO) TableName() string { return "notifications" }

type NotificationTemplatePO struct {
	ID        uint64   `gorm:"column:id;primaryKey;autoIncrement"`
	Code      string   `gorm:"column:code;type:varchar(50);uniqueIndex;not null"`
	Type      string   `gorm:"column:type;type:varchar(20);not null"`
	Title     string   `gorm:"column:title;type:varchar(255);not null"`
	Content   string   `gorm:"column:content;type:text"`
	Channels  string   `gorm:"column:channels;type:json"`
	IsActive  bool     `gorm:"column:is_active;default:true"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (NotificationTemplatePO) TableName() string { return "notification_templates" }

type UserNotificationPreferencePO struct {
	ID         uint64   `gorm:"column:id;primaryKey;autoIncrement"`
	UserID     uint64   `gorm:"column:user_id;index;not null"`
	Type       string   `gorm:"column:type;type:varchar(20);not null"`
	Channels   string   `gorm:"column:channels;type:json"`
	Enabled    bool     `gorm:"column:enabled;default:true"`
	QuietStart int      `gorm:"column:quiet_start;default:0"`
	QuietEnd   int      `gorm:"column:quiet_end;default:0"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserNotificationPreferencePO) TableName() string { return "user_notification_preferences" }

type GormNotificationRepository struct {
	db *gorm.DB
}

func NewGormNotificationRepository(db *gorm.DB) *GormNotificationRepository {
	return &GormNotificationRepository{db: db}
}

func (r *GormNotificationRepository) Save(ctx context.Context, n *domain.Notification) error {
	po := toNotificationPO(n)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *GormNotificationRepository) Update(ctx context.Context, n *domain.Notification) error {
	po := toNotificationPO(n)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormNotificationRepository) GetByID(ctx context.Context, id uint64) (*domain.Notification, error) {
	var po NotificationPO
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toNotification(&po), nil
}

func (r *GormNotificationRepository) GetByNotificationID(ctx context.Context, notificationID string) (*domain.Notification, error) {
	var po NotificationPO
	err := r.db.WithContext(ctx).Where("notification_id = ?", notificationID).First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toNotification(&po), nil
}

func (r *GormNotificationRepository) ListByUserID(ctx context.Context, userID uint64, status domain.NotificationStatus, page, pageSize int) ([]*domain.Notification, int64, error) {
	var pos []*NotificationPO
	var total int64

	query := r.db.WithContext(ctx).Model(&NotificationPO{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	notifications := make([]*domain.Notification, len(pos))
	for i, po := range pos {
		notifications[i] = toNotification(po)
	}

	return notifications, total, nil
}

func (r *GormNotificationRepository) ListPending(ctx context.Context, limit int) ([]*domain.Notification, error) {
	var pos []*NotificationPO
	err := r.db.WithContext(ctx).
		Where("status = ? AND (scheduled_at IS NULL OR scheduled_at <= ?)", "PENDING", time.Now()).
		Order("priority DESC, created_at ASC").
		Limit(limit).
		Find(&pos).Error
	if err != nil {
		return nil, err
	}

	notifications := make([]*domain.Notification, len(pos))
	for i, po := range pos {
		notifications[i] = toNotification(po)
	}
	return notifications, nil
}

func (r *GormNotificationRepository) ListScheduled(ctx context.Context, before time.Time, limit int) ([]*domain.Notification, error) {
	var pos []*NotificationPO
	err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?", "PENDING", before).
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&pos).Error
	if err != nil {
		return nil, err
	}

	notifications := make([]*domain.Notification, len(pos))
	for i, po := range pos {
		notifications[i] = toNotification(po)
	}
	return notifications, nil
}

func (r *GormNotificationRepository) MarkBatchSent(ctx context.Context, ids []uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&NotificationPO{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"status":  "SENT",
			"sent_at": now,
		}).Error
}

func (r *GormNotificationRepository) MarkBatchFailed(ctx context.Context, ids []uint64, reason string) error {
	return r.db.WithContext(ctx).Model(&NotificationPO{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"status":      "FAILED",
			"fail_reason": reason,
		}).Error
}

func toNotificationPO(n *domain.Notification) *NotificationPO {
	return &NotificationPO{
		ID:             n.ID,
		NotificationID: n.NotificationID,
		UserID:         n.UserID,
		Type:           string(n.Type),
		Priority:       string(n.Priority),
		Title:          n.Title,
		Content:        n.Content,
		Status:         string(n.Status),
		SentAt:         n.SentAt,
		DeliveredAt:    n.DeliveredAt,
		ReadAt:         n.ReadAt,
		FailReason:     n.FailReason,
		RetryCount:     n.RetryCount,
		MaxRetries:     n.MaxRetries,
		ScheduledAt:    n.ScheduledAt,
		ExpiresAt:      n.ExpiresAt,
		CreatedAt:      n.CreatedAt,
		UpdatedAt:      n.UpdatedAt,
	}
}

func toNotification(po *NotificationPO) *domain.Notification {
	return &domain.Notification{
		ID:             po.ID,
		NotificationID: po.NotificationID,
		UserID:         po.UserID,
		Type:           domain.NotificationType(po.Type),
		Priority:       domain.NotificationPriority(po.Priority),
		Title:          po.Title,
		Content:        po.Content,
		Data:           make(map[string]any),
		Channels:       make([]domain.NotificationChannel, 0),
		Status:         domain.NotificationStatus(po.Status),
		SentAt:         po.SentAt,
		DeliveredAt:    po.DeliveredAt,
		ReadAt:         po.ReadAt,
		FailReason:     po.FailReason,
		RetryCount:     po.RetryCount,
		MaxRetries:     po.MaxRetries,
		ScheduledAt:    po.ScheduledAt,
		ExpiresAt:      po.ExpiresAt,
		CreatedAt:      po.CreatedAt,
		UpdatedAt:      po.UpdatedAt,
	}
}
