package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wyfcoding/pkg/idgen"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
)

// NotificationCommand 处理通知相关的命令操作
type NotificationCommand struct {
	repo           domain.NotificationRepository
	emailSender    domain.Sender
	smsSender      domain.Sender
	eventPublisher domain.EventPublisher
}

// NewNotificationCommand 创建新的 NotificationCommand 实例
func NewNotificationCommand(
	repo domain.NotificationRepository,
	emailSender domain.Sender,
	smsSender domain.Sender,
	eventPublisher domain.EventPublisher,
) *NotificationCommand {
	return &NotificationCommand{
		repo:           repo,
		emailSender:    emailSender,
		smsSender:      smsSender,
		eventPublisher: eventPublisher,
	}
}

// SendNotification 发送通知
func (c *NotificationCommand) SendNotification(ctx context.Context, cmd SendNotificationCommand) (string, error) {
	notification := &domain.Notification{
		NotificationID: fmt.Sprintf("%d", idgen.GenID()),
		UserID:         cmd.UserID,
		Channel:        domain.Channel(cmd.Channel),
		Subject:        cmd.Subject,
		Content:        cmd.Content,
		Recipient:      cmd.Recipient,
		Status:         domain.StatusPending,
	}

	if err := c.repo.Save(ctx, notification); err != nil {
		return "", err
	}

	// 发布通知创建事件
	createdEvent := domain.NotificationCreatedEvent{
		NotificationID: notification.NotificationID,
		UserID:         notification.UserID,
		Channel:        notification.Channel,
		Recipient:      notification.Recipient,
		Subject:        notification.Subject,
		Content:        notification.Content,
		Status:         notification.Status,
		OccurredOn:     time.Now(),
	}

	if err := c.eventPublisher.PublishNotificationCreated(createdEvent); err != nil {
		// 记录错误但不中断流程
	}

	var err error
	switch notification.Channel {
	case domain.ChannelEmail:
		err = c.emailSender.Send(ctx, cmd.Recipient, cmd.Subject, cmd.Content)
	case domain.ChannelSMS:
		err = c.smsSender.Send(ctx, cmd.Recipient, cmd.Subject, cmd.Content)
	default:
		// Fallback or ignore
	}

	oldStatus := notification.Status
	if err != nil {
		notification.Status = domain.StatusFailed
		notification.ErrorMsg = err.Error()

		// 发布通知发送失败事件
		failedEvent := domain.NotificationFailedEvent{
			NotificationID: notification.NotificationID,
			UserID:         notification.UserID,
			Channel:        notification.Channel,
			Recipient:      notification.Recipient,
			ErrorMsg:       err.Error(),
			FailedAt:       time.Now().Unix(),
			OccurredOn:     time.Now(),
		}

		if err := c.eventPublisher.PublishNotificationFailed(failedEvent); err != nil {
			// 记录错误但不中断流程
		}
	} else {
		notification.Status = domain.StatusSent

		// 发布通知发送成功事件
		sentEvent := domain.NotificationSentEvent{
			NotificationID: notification.NotificationID,
			UserID:         notification.UserID,
			Channel:        notification.Channel,
			Recipient:      notification.Recipient,
			SentAt:         time.Now().Unix(),
			OccurredOn:     time.Now(),
		}

		if err := c.eventPublisher.PublishNotificationSent(sentEvent); err != nil {
			// 记录错误但不中断流程
		}
	}

	if err := c.repo.Save(ctx, notification); err != nil {
		// 记录错误但不中断流程
	}

	// 发布通知状态变更事件
	statusChangedEvent := domain.NotificationStatusChangedEvent{
		NotificationID: notification.NotificationID,
		OldStatus:      oldStatus,
		NewStatus:      notification.Status,
		UpdatedAt:      time.Now().Unix(),
		OccurredOn:     time.Now(),
	}

	if err := c.eventPublisher.PublishNotificationStatusChanged(statusChangedEvent); err != nil {
		// 记录错误但不中断流程
	}

	return notification.NotificationID, nil
}

// SendBatchNotification 发送批量通知
func (c *NotificationCommand) SendBatchNotification(ctx context.Context, cmd BatchSendNotificationCommand) ([]string, error) {
	notificationIDs := make([]string, 0, len(cmd.Recipients))
	userIDs := make([]string, 0, len(cmd.Recipients))

	for _, recipient := range cmd.Recipients {
		notification := &domain.Notification{
			NotificationID: fmt.Sprintf("%d", idgen.GenID()),
			UserID:         cmd.UserID,
			Channel:        domain.Channel(cmd.Channel),
			Subject:        cmd.Subject,
			Content:        cmd.Content,
			Recipient:      recipient,
			Status:         domain.StatusPending,
		}

		if err := c.repo.Save(ctx, notification); err != nil {
			continue
		}

		notificationIDs = append(notificationIDs, notification.NotificationID)
		userIDs = append(userIDs, notification.UserID)

		// 发布通知创建事件
		createdEvent := domain.NotificationCreatedEvent{
			NotificationID: notification.NotificationID,
			UserID:         notification.UserID,
			Channel:        notification.Channel,
			Recipient:      notification.Recipient,
			Subject:        notification.Subject,
			Content:        notification.Content,
			Status:         notification.Status,
			OccurredOn:     time.Now(),
		}

		if err := c.eventPublisher.PublishNotificationCreated(createdEvent); err != nil {
			// 记录错误但不中断流程
		}
	}

	// 发布批量通知创建事件
	batchEvent := domain.BatchNotificationCreatedEvent{
		BatchID:         fmt.Sprintf("%d", idgen.GenID()),
		NotificationIDs: notificationIDs,
		UserIDs:         userIDs,
		Channel:         domain.Channel(cmd.Channel),
		Count:           len(notificationIDs),
		CreatedAt:       time.Now().Unix(),
		OccurredOn:      time.Now(),
	}

	if err := c.eventPublisher.PublishBatchNotificationCreated(batchEvent); err != nil {
		// 记录错误但不中断流程
	}

	return notificationIDs, nil
}
