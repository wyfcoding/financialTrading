// 包 基础设施层实现
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
)

// NotificationModel 通知数据库模型
// 对应数据库中的 notifications 表
type NotificationModel struct {
	gorm.Model
	ID           string `gorm:"column:id;type:varchar(36);primaryKey;comment:通知ID"`
	UserID       string `gorm:"column:user_id;type:varchar(36);index;not null;comment:用户ID"`
	Type         string `gorm:"column:type;type:varchar(20);not null;comment:通知类型"`
	Subject      string `gorm:"column:subject;type:varchar(255);comment:主题"`
	Content      string `gorm:"column:content;type:text;comment:内容"`
	Target       string `gorm:"column:target;type:varchar(255);not null;comment:发送目标"`
	Status       string `gorm:"column:status;type:varchar(20);default:'PENDING';index;comment:状态"`
	ErrorMessage string `gorm:"column:error_message;type:text;comment:错误信息"`
}

// 指定表名
func (NotificationModel) TableName() string {
	return "notifications"
}

// 将数据库模型转换为领域实体
func (m *NotificationModel) ToDomain() *domain.Notification {
	return &domain.Notification{
		Model:        m.Model,
		ID:           m.ID,
		UserID:       m.UserID,
		Type:         domain.NotificationType(m.Type),
		Subject:      m.Subject,
		Content:      m.Content,
		Target:       m.Target,
		Status:       domain.NotificationStatus(m.Status),
		ErrorMessage: m.ErrorMessage,
		CreatedAt:    m.CreatedAt,
		SentAt:       m.UpdatedAt, // 简化处理
	}
}

// NotificationRepositoryImpl 通知仓储实现
type NotificationRepositoryImpl struct {
	db *gorm.DB
}

// NewNotificationRepository 创建通知仓储实例
func NewNotificationRepository(db *gorm.DB) domain.NotificationRepository {
	return &NotificationRepositoryImpl{db: db}
}

// Save 保存通知
func (r *NotificationRepositoryImpl) Save(ctx context.Context, notification *domain.Notification) error {
	model := &NotificationModel{
		Model:        notification.Model,
		ID:           notification.ID,
		UserID:       notification.UserID,
		Type:         string(notification.Type),
		Subject:      notification.Subject,
		Content:      notification.Content,
		Target:       notification.Target,
		Status:       string(notification.Status),
		ErrorMessage: notification.ErrorMessage,
	}

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		logging.Error(ctx, "Failed to save notification",
			"notification_id", notification.ID,
			"error", err,
		)
		return fmt.Errorf("failed to save notification: %w", err)
	}

	notification.Model = model.Model
	return nil
}

// GetByID 根据 ID 获取通知
func (r *NotificationRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	var model NotificationModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get notification",
			"notification_id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}
	return model.ToDomain(), nil
}

// ListByUserID 获取用户的通知列表
func (r *NotificationRepositoryImpl) ListByUserID(ctx context.Context, userID string, limit int, offset int) ([]*domain.Notification, error) {
	var models []NotificationModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Limit(limit).Offset(offset).Order("created_at desc").Find(&models).Error; err != nil {
		logging.Error(ctx, "Failed to list notifications by user",
			"user_id", userID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to list notifications by user: %w", err)
	}

	result := make([]*domain.Notification, len(models))
	for i, m := range models {
		result[i] = m.ToDomain()
	}
	return result, nil
}
