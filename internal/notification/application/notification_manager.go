package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
)

// NotificationManager 处理所有通知相关的写入操作（Commands）。
type NotificationManager struct {
	repo        domain.NotificationRepository
	emailSender domain.Sender
	smsSender   domain.Sender
}

// NewNotificationManager 构造函数。
func NewNotificationManager(repo domain.NotificationRepository, emailSender domain.Sender, smsSender domain.Sender) *NotificationManager {
	return &NotificationManager{
		repo:        repo,
		emailSender: emailSender,
		smsSender:   smsSender,
	}
}

// SendNotification 发送通知
func (m *NotificationManager) SendNotification(ctx context.Context, userID string, notificationType string, subject string, content string, target string) (string, error) {
	notification := &domain.Notification{
		NotificationID: fmt.Sprintf("%d", idgen.GenID()),
		UserID:         userID,
		Type:           domain.NotificationType(notificationType),
		Subject:        subject,
		Content:        content,
		Target:         target,
		Status:         domain.NotificationStatusPending,
	}

	if err := m.repo.Save(ctx, notification); err != nil {
		return "", err
	}

	var err error
	switch notification.Type {
	case domain.NotificationTypeEmail:
		err = m.emailSender.Send(ctx, target, subject, content)
	case domain.NotificationTypeSMS:
		err = m.smsSender.Send(ctx, target, subject, content)
	default:
		err = fmt.Errorf("unsupported notification type: %s", notificationType)
	}

	if err != nil {
		notification.Status = domain.NotificationStatusFailed
		notification.ErrorMessage = err.Error()
	} else {
		notification.Status = domain.NotificationStatusSent
		now := time.Now()
		notification.SentAt = &now
	}
	if err := m.repo.Save(ctx, notification); err != nil {
		logging.Error(ctx, "NotificationManager: failed to save notification", "error", err)
	}

	return notification.NotificationID, nil
}
