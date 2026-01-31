package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
	"gorm.io/gorm"
)

type notificationRepository struct{ db *gorm.DB }

func NewNotificationRepository(db *gorm.DB) domain.NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Save(ctx context.Context, n *domain.Notification) error {
	return r.db.WithContext(ctx).Save(n).Error
}

func (r *notificationRepository) Get(ctx context.Context, id string) (*domain.Notification, error) {
	var n domain.Notification
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&n).Error
	return &n, err
}

func (r *notificationRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, int64, error) {
	var notifications []*domain.Notification
	var total int64
	r.db.WithContext(ctx).Model(&domain.Notification{}).Where("user_id = ?", userID).Count(&total)
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Offset(offset).Limit(limit).Find(&notifications).Error
	return notifications, total, err
}
