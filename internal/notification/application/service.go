package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/wyfcoding/financialtrading/internal/notification/domain"
)

type SendNotificationCommand struct {
	UserID     uint64
	Type       domain.NotificationType
	Priority   domain.NotificationPriority
	Title      string
	Content    string
	Data       map[string]any
	Channels   []domain.NotificationChannel
}

type SendNotificationResult struct {
	NotificationID string
	Status         domain.NotificationStatus
}

type SendBatchNotificationCommand struct {
	UserIDs   []uint64
	Type      domain.NotificationType
	Priority  domain.NotificationPriority
	Title     string
	Content   string
	Data      map[string]any
	Channels  []domain.NotificationChannel
}

type NotificationApplicationService struct {
	notificationRepo domain.NotificationRepository
	templateRepo     domain.NotificationTemplateRepository
	preferenceRepo   domain.UserNotificationPreferenceRepository
	batchRepo        domain.NotificationBatchRepository
	emailSender      domain.EmailSender
	smsSender        domain.SMSSender
	pushSender       domain.PushSender
	wsSender         domain.WebSocketSender
	webhookSender    domain.WebhookSender
	logger           *slog.Logger
}

func NewNotificationApplicationService(
	notificationRepo domain.NotificationRepository,
	templateRepo domain.NotificationTemplateRepository,
	preferenceRepo domain.UserNotificationPreferenceRepository,
	batchRepo domain.NotificationBatchRepository,
	emailSender domain.EmailSender,
	smsSender domain.SMSSender,
	pushSender domain.PushSender,
	wsSender domain.WebSocketSender,
	webhookSender domain.WebhookSender,
	logger *slog.Logger,
) *NotificationApplicationService {
	return &NotificationApplicationService{
		notificationRepo: notificationRepo,
		templateRepo:     templateRepo,
		preferenceRepo:   preferenceRepo,
		batchRepo:        batchRepo,
		emailSender:      emailSender,
		smsSender:        smsSender,
		pushSender:       pushSender,
		wsSender:         wsSender,
		webhookSender:    webhookSender,
		logger:           logger,
	}
}

func (s *NotificationApplicationService) Send(ctx context.Context, cmd *SendNotificationCommand) (*SendNotificationResult, error) {
	notificationID := fmt.Sprintf("NT%s", uuid.New().String()[:16])
	notification := domain.NewNotification(notificationID, cmd.UserID, cmd.Type, cmd.Priority, cmd.Title, cmd.Content)
	
	if cmd.Data != nil {
		notification.SetData(cmd.Data)
	}

	pref, err := s.preferenceRepo.GetByUserIDAndType(ctx, cmd.UserID, cmd.Type)
	if err != nil {
		s.logger.Warn("failed to get notification preference", "error", err)
	}
	
	if pref != nil && !pref.Enabled {
		s.logger.Info("notification disabled for user", "user_id", cmd.UserID, "type", cmd.Type)
		return &SendNotificationResult{NotificationID: notificationID, Status: domain.StatusPending}, nil
	}

	if pref != nil && pref.QuietHours != nil && pref.QuietHours.IsQuietTime(time.Now()) {
		s.logger.Info("quiet hours active, notification will be delayed", "user_id", cmd.UserID)
		scheduledTime := s.calculateNextActiveTime(pref.QuietHours)
		notification.SetSchedule(&scheduledTime, nil)
	}

	if len(cmd.Channels) > 0 {
		for _, ch := range cmd.Channels {
			notification.AddChannel(ch)
		}
	} else if pref != nil && len(pref.Channels) > 0 {
		for _, ch := range pref.Channels {
			notification.AddChannel(ch)
		}
	} else {
		notification.AddChannel(domain.ChannelInApp)
	}

	if err := s.notificationRepo.Save(ctx, notification); err != nil {
		return nil, err
	}

	go s.sendNotification(context.Background(), notification)

	return &SendNotificationResult{
		NotificationID: notificationID,
		Status:         notification.Status,
	}, nil
}

func (s *NotificationApplicationService) SendBatch(ctx context.Context, cmd *SendBatchNotificationCommand) (string, error) {
	batchID := fmt.Sprintf("NB%s", uuid.New().String()[:16])
	batch := &domain.NotificationBatch{
		BatchID:    batchID,
		Type:       cmd.Type,
		Title:      cmd.Title,
		Content:    cmd.Content,
		UserIDs:    cmd.UserIDs,
		TotalCount: len(cmd.UserIDs),
		Status:     "PROCESSING",
		CreatedAt:  time.Now(),
	}

	if err := s.batchRepo.Save(ctx, batch); err != nil {
		return "", err
	}

	go s.processBatch(context.Background(), batch, cmd)

	return batchID, nil
}

func (s *NotificationApplicationService) processBatch(ctx context.Context, batch *domain.NotificationBatch, cmd *SendBatchNotificationCommand) {
	for _, userID := range batch.UserIDs {
		_, err := s.Send(ctx, &SendNotificationCommand{
			UserID:   userID,
			Type:     cmd.Type,
			Priority: cmd.Priority,
			Title:    cmd.Title,
			Content:  cmd.Content,
			Data:     cmd.Data,
			Channels: cmd.Channels,
		})
		if err != nil {
			batch.FailCount++
			s.logger.Error("failed to send notification in batch", "user_id", userID, "error", err)
		} else {
			batch.SentCount++
		}
	}

	now := time.Now()
	batch.Status = "COMPLETED"
	batch.CompletedAt = &now
	s.batchRepo.Update(ctx, batch)
}

func (s *NotificationApplicationService) sendNotification(ctx context.Context, notification *domain.Notification) {
	for _, channel := range notification.Channels {
		var err error
		
		switch channel {
		case domain.ChannelEmail:
			err = s.emailSender.Send(ctx, fmt.Sprintf("user_%d@example.com", notification.UserID), notification.Title, notification.Content, notification.Data)
		case domain.ChannelSMS:
			err = s.smsSender.Send(ctx, fmt.Sprintf("+1234567890%d", notification.UserID), notification.Content)
		case domain.ChannelPush:
			err = s.pushSender.Send(ctx, notification.UserID, notification.Title, notification.Content, notification.Data)
		case domain.ChannelInApp, domain.ChannelWebSocket:
			err = s.wsSender.Broadcast(ctx, notification.UserID, notification)
		case domain.ChannelWebhook:
			webhookURL := fmt.Sprintf("https://api.example.com/users/%d/webhook", notification.UserID)
			err = s.webhookSender.Send(ctx, webhookURL, notification)
		}

		if err != nil {
			s.logger.Error("failed to send notification", "notification_id", notification.NotificationID, "channel", channel, "error", err)
			notification.MarkFailed(err.Error())
		} else {
			notification.MarkSent()
			s.logger.Info("notification sent", "notification_id", notification.NotificationID, "channel", channel)
		}
	}

	if notification.Status == domain.StatusSent {
		notification.MarkDelivered()
	}

	s.notificationRepo.Update(ctx, notification)
}

func (s *NotificationApplicationService) calculateNextActiveTime(qh *domain.QuietHours) time.Time {
	now := time.Now()
	endTime := time.Date(now.Year(), now.Month(), now.Day(), qh.EndHour, qh.EndMin, 0, 0, now.Location())
	if now.After(endTime) {
		endTime = endTime.Add(24 * time.Hour)
	}
	return endTime
}

func (s *NotificationApplicationService) GetNotification(ctx context.Context, notificationID string) (*domain.Notification, error) {
	return s.notificationRepo.GetByNotificationID(ctx, notificationID)
}

func (s *NotificationApplicationService) ListNotifications(ctx context.Context, userID uint64, status domain.NotificationStatus, page, pageSize int) ([]*domain.Notification, int64, error) {
	return s.notificationRepo.ListByUserID(ctx, userID, status, page, pageSize)
}

func (s *NotificationApplicationService) MarkAsRead(ctx context.Context, notificationID string) error {
	notification, err := s.notificationRepo.GetByNotificationID(ctx, notificationID)
	if err != nil {
		return err
	}
	if notification == nil {
		return domain.ErrNotificationNotFound
	}
	notification.MarkRead()
	return s.notificationRepo.Update(ctx, notification)
}

func (s *NotificationApplicationService) ProcessPendingNotifications(ctx context.Context) error {
	notifications, err := s.notificationRepo.ListPending(ctx, 100)
	if err != nil {
		return err
	}

	for _, n := range notifications {
		if n.IsExpired() {
			n.MarkFailed("expired")
			s.notificationRepo.Update(ctx, n)
			continue
		}
		go s.sendNotification(context.Background(), n)
	}

	return nil
}

func (s *NotificationApplicationService) ProcessScheduledNotifications(ctx context.Context) error {
	notifications, err := s.notificationRepo.ListScheduled(ctx, time.Now(), 100)
	if err != nil {
		return err
	}

	for _, n := range notifications {
		go s.sendNotification(context.Background(), n)
	}

	return nil
}

func (s *NotificationApplicationService) SetPreference(ctx context.Context, userID uint64, notifType domain.NotificationType, channels []domain.NotificationChannel, enabled bool) error {
	pref, err := s.preferenceRepo.GetByUserIDAndType(ctx, userID, notifType)
	if err != nil {
		pref = domain.NewUserNotificationPreference(userID, notifType)
	}
	
	pref.SetChannels(channels)
	pref.SetEnabled(enabled)
	
	if pref.ID == 0 {
		return s.preferenceRepo.Save(ctx, pref)
	}
	return s.preferenceRepo.Save(ctx, pref)
}

func (s *NotificationApplicationService) GetPreferences(ctx context.Context, userID uint64) ([]*domain.UserNotificationPreference, error) {
	return s.preferenceRepo.GetByUserID(ctx, userID)
}

func (s *NotificationApplicationService) SendFromTemplate(ctx context.Context, templateCode string, userID uint64, data map[string]any) (*SendNotificationResult, error) {
	template, err := s.templateRepo.GetByCode(ctx, templateCode)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, domain.ErrTemplateNotFound
	}

	return s.Send(ctx, &SendNotificationCommand{
		UserID:   userID,
		Type:     template.Type,
		Priority: domain.PriorityNormal,
		Title:    template.Title,
		Content:  template.Content,
		Data:     data,
		Channels: template.Channels,
	})
}
