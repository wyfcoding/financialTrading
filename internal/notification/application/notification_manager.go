package application

import (
	"context"
	"fmt"

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
func (m *NotificationManager) SendNotification(ctx context.Context, cmd SendNotificationCommand) (string, error) {
	notification := &domain.Notification{
		NotificationID: fmt.Sprintf("%d", idgen.GenID()),
		UserID:         cmd.UserID,
		Channel:        domain.Channel(cmd.Channel),
		Subject:        cmd.Subject,
		Content:        cmd.Content,
		Recipient:      cmd.Recipient,
		Status:         domain.StatusPending,
	}

	if err := m.repo.Save(ctx, notification); err != nil {
		return "", err
	}

	var err error
	switch notification.Channel {
	case domain.ChannelEmail:
		err = m.emailSender.Send(ctx, cmd.Recipient, cmd.Subject, cmd.Content)
	case domain.ChannelSMS:
		err = m.smsSender.Send(ctx, cmd.Recipient, cmd.Subject, cmd.Content)
	default:
		// Fallback or ignore
	}

	if err != nil {
		notification.Status = domain.StatusFailed
		notification.ErrorMsg = err.Error()
	} else {
		notification.Status = domain.StatusSent
	}
	if err := m.repo.Save(ctx, notification); err != nil {
		logging.Error(ctx, "NotificationManager: failed to save notification", "error", err)
	}

	return notification.NotificationID, nil
}

// GetHistory 获取通知历史
func (m *NotificationManager) GetHistory(ctx context.Context, userID string, limit int) ([]*domain.Notification, error) {
	// ListByUserID returns (list, count, err)
	list, _, err := m.repo.ListByUserID(ctx, userID, limit, 0)
	return list, err
}
