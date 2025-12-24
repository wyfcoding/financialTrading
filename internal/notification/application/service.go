// 包 通知服务的用例逻辑
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wyfcoding/financialtrading/internal/notification/domain"
	"github.com/wyfcoding/pkg/logging"
)

// NotificationService 通知应用服务
// 负责处理通知的创建、发送和历史查询
type NotificationService struct {
	repo        domain.NotificationRepository // 通知仓储接口
	emailSender domain.Sender                 // 邮件发送器
	smsSender   domain.Sender                 // 短信发送器
}

// NewNotificationService 创建通知应用服务实例
// repo: 注入的通知仓储实现
// emailSender: 注入的邮件发送器实现
// smsSender: 注入的短信发送器实现
func NewNotificationService(repo domain.NotificationRepository, emailSender domain.Sender, smsSender domain.Sender) *NotificationService {
	return &NotificationService{
		repo:        repo,
		emailSender: emailSender,
		smsSender:   smsSender,
	}
}

// SendNotification 发送通知
func (s *NotificationService) SendNotification(ctx context.Context, userID string, notificationType string, subject string, content string, target string) (string, error) {
	// 1. 创建通知实体
	notification := &domain.Notification{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      domain.NotificationType(notificationType),
		Subject:   subject,
		Content:   content,
		Target:    target,
		Status:    domain.NotificationStatusPending,
		CreatedAt: time.Now(),
	}

	// 2. 保存到数据库
	if err := s.repo.Save(ctx, notification); err != nil {
		logging.Error(ctx, "Failed to save notification",
			"user_id", userID,
			"error", err,
		)
		return "", fmt.Errorf("failed to save notification: %w", err)
	}

	// 3. 发送通知（这里简化为同步发送，实际应异步）
	var err error
	switch notification.Type {
	case domain.NotificationTypeEmail:
		err = s.emailSender.Send(ctx, target, subject, content)
	case domain.NotificationTypeSMS:
		err = s.smsSender.Send(ctx, target, subject, content)
	default:
		err = fmt.Errorf("unsupported notification type: %s", notificationType)
	}

	// 4. 更新状态
	if err != nil {
		notification.Status = domain.NotificationStatusFailed
		notification.ErrorMessage = err.Error()
		logging.Error(ctx, "Failed to send notification",
			"notification_id", notification.ID,
			"error", err,
		)
	} else {
		notification.Status = domain.NotificationStatusSent
		notification.SentAt = time.Now()
		logging.Info(ctx, "Notification sent successfully",
			"notification_id", notification.ID,
			"type", notificationType,
		)
	}

	// 再次保存更新状态
	if err := s.repo.Save(ctx, notification); err != nil {
		logging.Error(ctx, "Failed to update notification status",
			"notification_id", notification.ID,
			"error", err,
		)
	}

	return notification.ID, nil
}

// GetNotificationHistory 获取通知历史
func (s *NotificationService) GetNotificationHistory(ctx context.Context, userID string, limit int, offset int) ([]*domain.Notification, error) {
	notifications, err := s.repo.ListByUserID(ctx, userID, limit, offset)
	if err != nil {
		logging.Error(ctx, "Failed to get notification history",
			"user_id", userID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get notification history: %w", err)
	}
	return notifications, nil
}
