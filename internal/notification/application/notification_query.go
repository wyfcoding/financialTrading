package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
)

// NotificationQuery 处理所有通知相关的查询操作（Queries）。
type NotificationQuery struct {
	repo domain.NotificationRepository
}

// NewNotificationQuery 构造函数。
func NewNotificationQuery(repo domain.NotificationRepository) *NotificationQuery {
	return &NotificationQuery{
		repo: repo,
	}
}

// GetNotificationHistory 获取通知历史
func (q *NotificationQuery) GetNotificationHistory(ctx context.Context, userID string, limit int, offset int) ([]*domain.Notification, int64, error) {
	return q.repo.ListByUserID(ctx, userID, limit, offset)
}
