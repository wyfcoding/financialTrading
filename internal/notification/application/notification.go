package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
)

// NotificationService 通知门面服务，整合 Manager 和 Query。
type NotificationService struct {
	manager *NotificationManager
	query   *NotificationQuery
}

// NewNotificationService 构造函数。
func NewNotificationService(repo domain.NotificationRepository, emailSender domain.Sender, smsSender domain.Sender) *NotificationService {
	return &NotificationService{
		manager: NewNotificationManager(repo, emailSender, smsSender),
		query:   NewNotificationQuery(repo),
	}
}

// --- Manager (Writes) ---

func (s *NotificationService) SendNotification(ctx context.Context, userID string, notificationType string, subject string, content string, target string) (string, error) {
	return s.manager.SendNotification(ctx, userID, notificationType, subject, content, target)
}

// --- Query (Reads) ---

func (s *NotificationService) GetNotificationHistory(ctx context.Context, userID string, limit int, offset int) ([]*domain.Notification, int64, error) {
	return s.query.GetNotificationHistory(ctx, userID, limit, offset)
}
