package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
	"github.com/wyfcoding/financialtrading/internal/notification/infrastructure/messaging"
	"gorm.io/gorm"
)

// mockEventPublisher 事件发布者的空实现
type mockEventPublisher struct{}

func (m *mockEventPublisher) PublishNotificationCreated(event domain.NotificationCreatedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishNotificationSent(event domain.NotificationSentEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishNotificationFailed(event domain.NotificationFailedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishNotificationStatusChanged(event domain.NotificationStatusChangedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishBatchNotificationCreated(event domain.BatchNotificationCreatedEvent) error {
	return nil
}

// NotificationService 通知服务门面，整合命令和查询服务
type NotificationService struct {
	command *NotificationCommand
	query   *NotificationQuery
}

// NewNotificationService 构造函数
func NewNotificationService(repo domain.NotificationRepository, db interface{}) (*NotificationService, error) {
	// 创建事件发布者
	var eventPublisher domain.EventPublisher
	if gormDB, ok := db.(*gorm.DB); ok {
		eventPublisher = messaging.NewOutboxEventPublisher(gormDB)
	} else {
		// 使用空实现作为降级方案
		eventPublisher = &mockEventPublisher{}
	}

	// 创建发送器（使用空实现）
	emailSender := &dummySender{}
	smsSender := &dummySender{}

	// 创建命令服务
	command := NewNotificationCommand(
		repo,
		emailSender,
		smsSender,
		eventPublisher,
	)

	// 创建查询服务
	query := NewNotificationQuery(repo)

	return &NotificationService{
		command: command,
		query:   query,
	}, nil
}

// --- Command (Writes) ---

// SendNotification 发送通知
func (s *NotificationService) SendNotification(ctx context.Context, cmd SendNotificationCommand) (string, error) {
	return s.command.SendNotification(ctx, cmd)
}

// SendBatchNotification 发送批量通知
func (s *NotificationService) SendBatchNotification(ctx context.Context, cmd BatchSendNotificationCommand) ([]string, error) {
	return s.command.SendBatchNotification(ctx, cmd)
}

// --- Query (Reads) ---

// GetNotificationHistory 获取通知历史
func (s *NotificationService) GetNotificationHistory(ctx context.Context, userID string, limit int, offset int) ([]*domain.Notification, int64, error) {
	return s.query.GetNotificationHistory(ctx, userID, limit, offset)
}

// dummySender 简单的发送器实现
type dummySender struct{}

// Send 发送通知
func (s *dummySender) Send(ctx context.Context, recipient, subject, content string) error {
	// 简单实现，仅记录日志
	return nil
}
