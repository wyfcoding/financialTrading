package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
	"gorm.io/gorm"
)

type NotificationPO struct {
	ID             uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	NotificationID string     `gorm:"column:notification_id;type:varchar(32);uniqueIndex;not null"`
	UserID         uint64     `gorm:"column:user_id;index;not null"`
	Type           string     `gorm:"column:type;type:varchar(20);not null"`
	Priority       string     `gorm:"column:priority;type:varchar(20);not null"`
	Title          string     `gorm:"column:title;type:varchar(255);not null"`
	Content        string     `gorm:"column:content;type:text"`
	Data           string     `gorm:"column:data;type:json"`
	Channels       string     `gorm:"column:channels;type:json"`
	Status         string     `gorm:"column:status;type:varchar(20);not null;default:'PENDING'"`
	SentAt         *time.Time `gorm:"column:sent_at"`
	DeliveredAt    *time.Time `gorm:"column:delivered_at"`
	ReadAt         *time.Time `gorm:"column:read_at"`
	FailReason     string     `gorm:"column:fail_reason;type:varchar(255)"`
	RetryCount     int        `gorm:"column:retry_count;default:0"`
	MaxRetries     int        `gorm:"column:max_retries;default:3"`
	ScheduledAt    *time.Time `gorm:"column:scheduled_at"`
	ExpiresAt      *time.Time `gorm:"column:expires_at"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (NotificationPO) TableName() string { return "notifications" }

type NotificationTemplatePO struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	Code      string    `gorm:"column:code;type:varchar(50);uniqueIndex;not null"`
	Type      string    `gorm:"column:type;type:varchar(20);not null"`
	Title     string    `gorm:"column:title;type:varchar(255);not null"`
	Content   string    `gorm:"column:content;type:text"`
	Channels  string    `gorm:"column:channels;type:json"`
	IsActive  bool      `gorm:"column:is_active;default:true"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (NotificationTemplatePO) TableName() string { return "notification_templates" }

type UserNotificationPreferencePO struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	UserID     uint64    `gorm:"column:user_id;index;not null"`
	Type       string    `gorm:"column:type;type:varchar(20);not null"`
	Channels   string    `gorm:"column:channels;type:json"`
	Enabled    bool      `gorm:"column:enabled;default:true"`
	QuietStart int       `gorm:"column:quiet_start;default:0"`
	QuietEnd   int       `gorm:"column:quiet_end;default:0"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserNotificationPreferencePO) TableName() string { return "user_notification_preferences" }

type NotificationBatchPO struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	BatchID     string     `gorm:"column:batch_id;type:varchar(32);uniqueIndex;not null"`
	Type        string     `gorm:"column:type;type:varchar(20);not null"`
	Title       string     `gorm:"column:title;type:varchar(255);not null"`
	Content     string     `gorm:"column:content;type:text"`
	UserIDs     string     `gorm:"column:user_ids;type:json"`
	TotalCount  int        `gorm:"column:total_count;not null"`
	SentCount   int        `gorm:"column:sent_count;not null;default:0"`
	FailCount   int        `gorm:"column:fail_count;not null;default:0"`
	Status      string     `gorm:"column:status;type:varchar(20);not null"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"`
	CompletedAt *time.Time `gorm:"column:completed_at"`
}

func (NotificationBatchPO) TableName() string { return "notification_batches" }

type GormNotificationRepository struct {
	db *gorm.DB
}

func NewGormNotificationRepository(db *gorm.DB) *GormNotificationRepository {
	return &GormNotificationRepository{db: db}
}

func (r *GormNotificationRepository) Save(ctx context.Context, n *domain.Notification) error {
	po := toNotificationPO(n)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *GormNotificationRepository) Update(ctx context.Context, n *domain.Notification) error {
	po := toNotificationPO(n)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormNotificationRepository) GetByID(ctx context.Context, id uint64) (*domain.Notification, error) {
	var po NotificationPO
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toNotification(&po), nil
}

func (r *GormNotificationRepository) GetByNotificationID(ctx context.Context, notificationID string) (*domain.Notification, error) {
	var po NotificationPO
	err := r.db.WithContext(ctx).Where("notification_id = ?", notificationID).First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toNotification(&po), nil
}

func (r *GormNotificationRepository) ListByUserID(ctx context.Context, userID uint64, status domain.NotificationStatus, page, pageSize int) ([]*domain.Notification, int64, error) {
	var pos []*NotificationPO
	var total int64

	query := r.db.WithContext(ctx).Model(&NotificationPO{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	notifications := make([]*domain.Notification, len(pos))
	for i, po := range pos {
		notifications[i] = toNotification(po)
	}

	return notifications, total, nil
}

func (r *GormNotificationRepository) ListPending(ctx context.Context, limit int) ([]*domain.Notification, error) {
	var pos []*NotificationPO
	err := r.db.WithContext(ctx).
		Where("status = ? AND (scheduled_at IS NULL OR scheduled_at <= ?)", "PENDING", time.Now()).
		Order("priority DESC, created_at ASC").
		Limit(limit).
		Find(&pos).Error
	if err != nil {
		return nil, err
	}

	notifications := make([]*domain.Notification, len(pos))
	for i, po := range pos {
		notifications[i] = toNotification(po)
	}
	return notifications, nil
}

func (r *GormNotificationRepository) ListScheduled(ctx context.Context, before time.Time, limit int) ([]*domain.Notification, error) {
	var pos []*NotificationPO
	err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?", "PENDING", before).
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&pos).Error
	if err != nil {
		return nil, err
	}

	notifications := make([]*domain.Notification, len(pos))
	for i, po := range pos {
		notifications[i] = toNotification(po)
	}
	return notifications, nil
}

func (r *GormNotificationRepository) MarkBatchSent(ctx context.Context, ids []uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&NotificationPO{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"status":  "SENT",
			"sent_at": now,
		}).Error
}

func (r *GormNotificationRepository) MarkBatchFailed(ctx context.Context, ids []uint64, reason string) error {
	return r.db.WithContext(ctx).Model(&NotificationPO{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"status":      "FAILED",
			"fail_reason": reason,
		}).Error
}

func toNotificationPO(n *domain.Notification) *NotificationPO {
	return &NotificationPO{
		ID:             n.ID,
		NotificationID: n.NotificationID,
		UserID:         n.UserID,
		Type:           string(n.Type),
		Priority:       string(n.Priority),
		Title:          n.Title,
		Content:        n.Content,
		Data:           mustJSON(n.Data),
		Channels:       mustJSON(n.Channels),
		Status:         string(n.Status),
		SentAt:         n.SentAt,
		DeliveredAt:    n.DeliveredAt,
		ReadAt:         n.ReadAt,
		FailReason:     n.FailReason,
		RetryCount:     n.RetryCount,
		MaxRetries:     n.MaxRetries,
		ScheduledAt:    n.ScheduledAt,
		ExpiresAt:      n.ExpiresAt,
		CreatedAt:      n.CreatedAt,
		UpdatedAt:      n.UpdatedAt,
	}
}

func toNotification(po *NotificationPO) *domain.Notification {
	var data map[string]any
	_ = json.Unmarshal([]byte(po.Data), &data)
	if data == nil {
		data = make(map[string]any)
	}

	var channels []domain.NotificationChannel
	_ = json.Unmarshal([]byte(po.Channels), &channels)
	if channels == nil {
		channels = make([]domain.NotificationChannel, 0)
	}

	return &domain.Notification{
		ID:             po.ID,
		NotificationID: po.NotificationID,
		UserID:         po.UserID,
		Type:           domain.NotificationType(po.Type),
		Priority:       domain.NotificationPriority(po.Priority),
		Title:          po.Title,
		Content:        po.Content,
		Data:           data,
		Channels:       channels,
		Status:         domain.NotificationStatus(po.Status),
		SentAt:         po.SentAt,
		DeliveredAt:    po.DeliveredAt,
		ReadAt:         po.ReadAt,
		FailReason:     po.FailReason,
		RetryCount:     po.RetryCount,
		MaxRetries:     po.MaxRetries,
		ScheduledAt:    po.ScheduledAt,
		ExpiresAt:      po.ExpiresAt,
		CreatedAt:      po.CreatedAt,
		UpdatedAt:      po.UpdatedAt,
	}
}

type GormNotificationTemplateRepository struct {
	db *gorm.DB
}

func NewGormNotificationTemplateRepository(db *gorm.DB) *GormNotificationTemplateRepository {
	return &GormNotificationTemplateRepository{db: db}
}

func (r *GormNotificationTemplateRepository) Save(ctx context.Context, template *domain.NotificationTemplate) error {
	po := toTemplatePO(template)
	if po.ID == 0 {
		return r.db.WithContext(ctx).Create(po).Error
	}
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormNotificationTemplateRepository) GetByCode(ctx context.Context, code string) (*domain.NotificationTemplate, error) {
	var po NotificationTemplatePO
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toTemplate(&po), nil
}

func (r *GormNotificationTemplateRepository) GetByType(ctx context.Context, notifType domain.NotificationType) ([]*domain.NotificationTemplate, error) {
	var pos []*NotificationTemplatePO
	err := r.db.WithContext(ctx).Where("type = ?", string(notifType)).Find(&pos).Error
	if err != nil {
		return nil, err
	}

	out := make([]*domain.NotificationTemplate, 0, len(pos))
	for _, po := range pos {
		out = append(out, toTemplate(po))
	}
	return out, nil
}

func (r *GormNotificationTemplateRepository) ListActive(ctx context.Context) ([]*domain.NotificationTemplate, error) {
	var pos []*NotificationTemplatePO
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&pos).Error
	if err != nil {
		return nil, err
	}

	out := make([]*domain.NotificationTemplate, 0, len(pos))
	for _, po := range pos {
		out = append(out, toTemplate(po))
	}
	return out, nil
}

type GormUserNotificationPreferenceRepository struct {
	db *gorm.DB
}

func NewGormUserNotificationPreferenceRepository(db *gorm.DB) *GormUserNotificationPreferenceRepository {
	return &GormUserNotificationPreferenceRepository{db: db}
}

func (r *GormUserNotificationPreferenceRepository) Save(ctx context.Context, pref *domain.UserNotificationPreference) error {
	po := toPreferencePO(pref)
	if po.ID == 0 {
		return r.db.WithContext(ctx).Create(po).Error
	}
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormUserNotificationPreferenceRepository) GetByUserIDAndType(ctx context.Context, userID uint64, notifType domain.NotificationType) (*domain.UserNotificationPreference, error) {
	var po UserNotificationPreferencePO
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND type = ?", userID, string(notifType)).
		First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrPreferenceNotFound
	}
	if err != nil {
		return nil, err
	}
	return toPreference(&po), nil
}

func (r *GormUserNotificationPreferenceRepository) GetByUserID(ctx context.Context, userID uint64) ([]*domain.UserNotificationPreference, error) {
	var pos []*UserNotificationPreferencePO
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&pos).Error
	if err != nil {
		return nil, err
	}

	out := make([]*domain.UserNotificationPreference, 0, len(pos))
	for _, po := range pos {
		out = append(out, toPreference(po))
	}
	return out, nil
}

type GormNotificationBatchRepository struct {
	db *gorm.DB
}

func NewGormNotificationBatchRepository(db *gorm.DB) *GormNotificationBatchRepository {
	return &GormNotificationBatchRepository{db: db}
}

func (r *GormNotificationBatchRepository) Save(ctx context.Context, batch *domain.NotificationBatch) error {
	po := toBatchPO(batch)
	if po.ID == 0 {
		return r.db.WithContext(ctx).Create(po).Error
	}
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormNotificationBatchRepository) Update(ctx context.Context, batch *domain.NotificationBatch) error {
	po := toBatchPO(batch)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormNotificationBatchRepository) GetByID(ctx context.Context, id uint64) (*domain.NotificationBatch, error) {
	var po NotificationBatchPO
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBatch(&po), nil
}

func (r *GormNotificationBatchRepository) GetByBatchID(ctx context.Context, batchID string) (*domain.NotificationBatch, error) {
	var po NotificationBatchPO
	err := r.db.WithContext(ctx).Where("batch_id = ?", batchID).First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBatch(&po), nil
}

type NoopEmailSender struct{}

func (NoopEmailSender) Send(_ context.Context, _, _, _ string, _ map[string]any) error { return nil }

type NoopSMSSender struct{}

func (NoopSMSSender) Send(_ context.Context, _, _ string) error { return nil }

type NoopPushSender struct{}

func (NoopPushSender) Send(_ context.Context, _ uint64, _, _ string, _ map[string]any) error {
	return nil
}

type NoopWebSocketSender struct{}

func (NoopWebSocketSender) Broadcast(_ context.Context, _ uint64, _ any) error { return nil }

type NoopWebhookSender struct{}

func (NoopWebhookSender) Send(_ context.Context, _ string, _ any) error { return nil }

func toTemplatePO(t *domain.NotificationTemplate) *NotificationTemplatePO {
	return &NotificationTemplatePO{
		ID:        t.ID,
		Code:      t.Code,
		Type:      string(t.Type),
		Title:     t.Title,
		Content:   t.Content,
		Channels:  mustJSON(t.Channels),
		IsActive:  t.IsActive,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func toTemplate(po *NotificationTemplatePO) *domain.NotificationTemplate {
	var channels []domain.NotificationChannel
	_ = json.Unmarshal([]byte(po.Channels), &channels)
	if channels == nil {
		channels = make([]domain.NotificationChannel, 0)
	}

	return &domain.NotificationTemplate{
		ID:        po.ID,
		Code:      po.Code,
		Type:      domain.NotificationType(po.Type),
		Title:     po.Title,
		Content:   po.Content,
		Channels:  channels,
		IsActive:  po.IsActive,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}
}

func toPreferencePO(p *domain.UserNotificationPreference) *UserNotificationPreferencePO {
	po := &UserNotificationPreferencePO{
		ID:        p.ID,
		UserID:    p.UserID,
		Type:      string(p.Type),
		Channels:  mustJSON(p.Channels),
		Enabled:   p.Enabled,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
	if p.QuietHours != nil {
		po.QuietStart = p.QuietHours.StartHour*60 + p.QuietHours.StartMin
		po.QuietEnd = p.QuietHours.EndHour*60 + p.QuietHours.EndMin
	}
	return po
}

func toPreference(po *UserNotificationPreferencePO) *domain.UserNotificationPreference {
	var channels []domain.NotificationChannel
	_ = json.Unmarshal([]byte(po.Channels), &channels)
	if channels == nil {
		channels = make([]domain.NotificationChannel, 0)
	}

	var quietHours *domain.QuietHours
	if po.QuietStart > 0 || po.QuietEnd > 0 {
		quietHours = &domain.QuietHours{
			StartHour: po.QuietStart / 60,
			StartMin:  po.QuietStart % 60,
			EndHour:   po.QuietEnd / 60,
			EndMin:    po.QuietEnd % 60,
		}
	}

	return &domain.UserNotificationPreference{
		ID:         po.ID,
		UserID:     po.UserID,
		Type:       domain.NotificationType(po.Type),
		Channels:   channels,
		Enabled:    po.Enabled,
		QuietHours: quietHours,
		CreatedAt:  po.CreatedAt,
		UpdatedAt:  po.UpdatedAt,
	}
}

func toBatchPO(b *domain.NotificationBatch) *NotificationBatchPO {
	return &NotificationBatchPO{
		ID:          b.ID,
		BatchID:     b.BatchID,
		Type:        string(b.Type),
		Title:       b.Title,
		Content:     b.Content,
		UserIDs:     mustJSON(b.UserIDs),
		TotalCount:  b.TotalCount,
		SentCount:   b.SentCount,
		FailCount:   b.FailCount,
		Status:      b.Status,
		CreatedAt:   b.CreatedAt,
		CompletedAt: b.CompletedAt,
	}
}

func toBatch(po *NotificationBatchPO) *domain.NotificationBatch {
	var userIDs []uint64
	_ = json.Unmarshal([]byte(po.UserIDs), &userIDs)
	if userIDs == nil {
		userIDs = make([]uint64, 0)
	}

	return &domain.NotificationBatch{
		ID:          po.ID,
		BatchID:     po.BatchID,
		Type:        domain.NotificationType(po.Type),
		Title:       po.Title,
		Content:     po.Content,
		UserIDs:     userIDs,
		TotalCount:  po.TotalCount,
		SentCount:   po.SentCount,
		FailCount:   po.FailCount,
		Status:      po.Status,
		CreatedAt:   po.CreatedAt,
		CompletedAt: po.CompletedAt,
	}
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}
