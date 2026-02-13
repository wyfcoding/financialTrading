package interfaces

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/notification/application"
	"github.com/wyfcoding/financialtrading/internal/notification/domain"
)

type NotificationHandler struct {
	app *application.NotificationApplicationService
}

func NewNotificationHandler(app *application.NotificationApplicationService) *NotificationHandler {
	return &NotificationHandler{app: app}
}

type SendNotificationRequest struct {
	UserID   uint64   `json:"user_id"`
	Type     string   `json:"type"`
	Priority string   `json:"priority"`
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Data     map[string]any `json:"data"`
	Channels []string `json:"channels"`
}

type SendNotificationResponse struct {
	NotificationID string `json:"notification_id"`
	Status         string `json:"status"`
}

func (h *NotificationHandler) SendNotification(ctx context.Context, req *SendNotificationRequest) (*SendNotificationResponse, error) {
	channels := make([]domain.NotificationChannel, len(req.Channels))
	for i, c := range req.Channels {
		channels[i] = domain.NotificationChannel(c)
	}

	result, err := h.app.Send(ctx, &application.SendNotificationCommand{
		UserID:   req.UserID,
		Type:     domain.NotificationType(req.Type),
		Priority: domain.NotificationPriority(req.Priority),
		Title:    req.Title,
		Content:  req.Content,
		Data:     req.Data,
		Channels: channels,
	})
	if err != nil {
		return nil, err
	}

	return &SendNotificationResponse{
		NotificationID: result.NotificationID,
		Status:         string(result.Status),
	}, nil
}

type ListNotificationsRequest struct {
	UserID   uint64 `json:"user_id"`
	Status   string `json:"status"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type ListNotificationsResponse struct {
	Notifications []*NotificationDTO `json:"notifications"`
	Total         int64              `json:"total"`
	Page          int                `json:"page"`
	PageSize      int                `json:"page_size"`
}

type NotificationDTO struct {
	ID             uint64    `json:"id"`
	NotificationID string    `json:"notification_id"`
	UserID         uint64    `json:"user_id"`
	Type           string    `json:"type"`
	Priority       string    `json:"priority"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	Status         string    `json:"status"`
	CreatedAt      string    `json:"created_at"`
}

func (h *NotificationHandler) ListNotifications(ctx context.Context, req *ListNotificationsRequest) (*ListNotificationsResponse, error) {
	notifications, total, err := h.app.ListNotifications(ctx, req.UserID, domain.NotificationStatus(req.Status), req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	dtos := make([]*NotificationDTO, len(notifications))
	for i, n := range notifications {
		dtos[i] = &NotificationDTO{
			ID:             n.ID,
			NotificationID: n.NotificationID,
			UserID:         n.UserID,
			Type:           string(n.Type),
			Priority:       string(n.Priority),
			Title:          n.Title,
			Content:        n.Content,
			Status:         string(n.Status),
			CreatedAt:      n.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return &ListNotificationsResponse{
		Notifications: dtos,
		Total:         total,
		Page:          req.Page,
		PageSize:      req.PageSize,
	}, nil
}

type MarkAsReadRequest struct {
	NotificationID string `json:"notification_id"`
}

func (h *NotificationHandler) MarkAsRead(ctx context.Context, req *MarkAsReadRequest) error {
	return h.app.MarkAsRead(ctx, req.NotificationID)
}

type SetPreferenceRequest struct {
	UserID   uint64   `json:"user_id"`
	Type     string   `json:"type"`
	Channels []string `json:"channels"`
	Enabled  bool     `json:"enabled"`
}

func (h *NotificationHandler) SetPreference(ctx context.Context, req *SetPreferenceRequest) error {
	channels := make([]domain.NotificationChannel, len(req.Channels))
	for i, c := range req.Channels {
		channels[i] = domain.NotificationChannel(c)
	}
	return h.app.SetPreference(ctx, req.UserID, domain.NotificationType(req.Type), channels, req.Enabled)
}

type GetPreferencesRequest struct {
	UserID uint64 `json:"user_id"`
}

type PreferenceDTO struct {
	ID       uint64   `json:"id"`
	UserID   uint64   `json:"user_id"`
	Type     string   `json:"type"`
	Channels []string `json:"channels"`
	Enabled  bool     `json:"enabled"`
}

func (h *NotificationHandler) GetPreferences(ctx context.Context, req *GetPreferencesRequest) ([]*PreferenceDTO, error) {
	prefs, err := h.app.GetPreferences(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*PreferenceDTO, len(prefs))
	for i, p := range prefs {
		channels := make([]string, len(p.Channels))
		for j, c := range p.Channels {
			channels[j] = string(c)
		}
		dtos[i] = &PreferenceDTO{
			ID:       p.ID,
			UserID:   p.UserID,
			Type:     string(p.Type),
			Channels: channels,
			Enabled:  p.Enabled,
		}
	}
	return dtos, nil
}
