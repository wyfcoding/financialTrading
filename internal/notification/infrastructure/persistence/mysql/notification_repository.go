// Package mysql 提供了通知仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NotificationModel 通知数据库模型
type NotificationModel struct {
	gorm.Model
	NotificationID string     `gorm:"column:notification_id;type:varchar(32);uniqueIndex;not null"`
	UserID         string     `gorm:"column:user_id;type:varchar(32);index;not null"`
	Type           string     `gorm:"column:type;type:varchar(20);not null"`
	Subject        string     `gorm:"column:subject;type:varchar(100)"`
	Content        string     `gorm:"column:content;type:text"`
	Target         string     `gorm:"column:target;type:varchar(100);not null"`
	Status         string     `gorm:"column:status;type:varchar(20);index;not null"`
	ErrorMessage   string     `gorm:"column:error_message;type:text"`
	SentAt         *time.Time `gorm:"column:sent_at;type:datetime"`
}

// TableName 指定表名
func (NotificationModel) TableName() string {
	return "notifications"
}

// notificationRepositoryImpl 是 domain.NotificationRepository 接口的 GORM 实现。
type notificationRepositoryImpl struct {
	db *gorm.DB
}

// NewNotificationRepository 创建通知仓储实例
func NewNotificationRepository(db *gorm.DB) domain.NotificationRepository {
	return &notificationRepositoryImpl{
		db: db,
	}
}

// Save 实现 domain.NotificationRepository.Save
func (r *notificationRepositoryImpl) Save(ctx context.Context, n *domain.Notification) error {
	m := &NotificationModel{
		Model:          n.Model,
		NotificationID: n.NotificationID,
		UserID:         n.UserID,
		Type:           string(n.Type),
		Subject:        n.Subject,
		Content:        n.Content,
		Target:         n.Target,
		Status:         string(n.Status),
		ErrorMessage:   n.ErrorMessage,
		SentAt:         n.SentAt,
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "notification_id"}},
		UpdateAll: true,
	}).Create(m).Error
	if err != nil {
		logging.Error(ctx, "notification_repository.Save failed", "notification_id", n.NotificationID, "error", err)
		return fmt.Errorf("failed to save notification: %w", err)
	}

	n.Model = m.Model
	return nil
}

// Get 实现 domain.NotificationRepository.Get
func (r *notificationRepositoryImpl) Get(ctx context.Context, notificationID string) (*domain.Notification, error) {
	var m NotificationModel
	if err := r.db.WithContext(ctx).Where("notification_id = ?", notificationID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "notification_repository.Get failed", "notification_id", notificationID, "error", err)
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return r.toDomain(&m), nil
}

// ListByUserID 实现 domain.NotificationRepository.ListByUserID
func (r *notificationRepositoryImpl) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, int64, error) {
	var ms []NotificationModel
	var total int64
	db := r.db.WithContext(ctx).Model(&NotificationModel{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&ms).Error; err != nil {
		logging.Error(ctx, "notification_repository.ListByUserID failed", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("failed to list notifications by user: %w", err)
	}

	res := make([]*domain.Notification, len(ms))
	for i, m := range ms {
		res[i] = r.toDomain(&m)
	}
	return res, total, nil
}

func (r *notificationRepositoryImpl) toDomain(m *NotificationModel) *domain.Notification {
	return &domain.Notification{
		Model:          m.Model,
		NotificationID: m.NotificationID,
		UserID:         m.UserID,
		Type:           domain.NotificationType(m.Type),
		Subject:        m.Subject,
		Content:        m.Content,
		Target:         m.Target,
		Status:         domain.NotificationStatus(m.Status),
		ErrorMessage:   m.ErrorMessage,
		SentAt:         m.SentAt,
	}
}
