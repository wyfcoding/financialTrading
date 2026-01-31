package messaging

import (
	"context"
	"encoding/json"
	"time"

	"github.com/wyfcoding/financialtrading/internal/user/domain"
	"gorm.io/gorm"
)

// OutboxMessage 消息队列
type OutboxMessage struct {
	ID        string    `gorm:"type:uuid;primary_key"`
	EventID   string    `gorm:"type:uuid;index"`
	EventType string    `gorm:"type:varchar(100);index"`
	Payload   string    `gorm:"type:text"`
	Status    string    `gorm:"type:varchar(20);index;default:'pending'"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
}

// TableName 指定表名
func (OutboxMessage) TableName() string {
	return "user_outbox_messages"
}

// OutboxPublisher 实现了 UserEventPublisher 接口，使用 Outbox 模式发布事件
type OutboxPublisher struct {
	db *gorm.DB
}

// NewOutboxPublisher 创建新的 OutboxPublisher
func NewOutboxPublisher(db *gorm.DB) *OutboxPublisher {
	return &OutboxPublisher{db: db}
}

// PublishUserCreated 发布用户创建事件
func (p *OutboxPublisher) PublishUserCreated(event domain.UserCreatedEvent) error {
	return p.publishEvent("UserCreatedEvent", event)
}

// PublishUserUpdated 发布用户更新事件
func (p *OutboxPublisher) PublishUserUpdated(event domain.UserUpdatedEvent) error {
	return p.publishEvent("UserUpdatedEvent", event)
}

// PublishUserDeleted 发布用户删除事件
func (p *OutboxPublisher) PublishUserDeleted(event domain.UserDeletedEvent) error {
	return p.publishEvent("UserDeletedEvent", event)
}

// PublishUserStatusChanged 发布用户状态变更事件
func (p *OutboxPublisher) PublishUserStatusChanged(event domain.UserStatusChangedEvent) error {
	return p.publishEvent("UserStatusChangedEvent", event)
}

// PublishUserRoleChanged 发布用户角色变更事件
func (p *OutboxPublisher) PublishUserRoleChanged(event domain.UserRoleChangedEvent) error {
	return p.publishEvent("UserRoleChangedEvent", event)
}

// PublishUserPasswordChanged 发布用户密码变更事件
func (p *OutboxPublisher) PublishUserPasswordChanged(event domain.UserPasswordChangedEvent) error {
	return p.publishEvent("UserPasswordChangedEvent", event)
}

// PublishUserEmailVerified 发布用户邮箱验证事件
func (p *OutboxPublisher) PublishUserEmailVerified(event domain.UserEmailVerifiedEvent) error {
	return p.publishEvent("UserEmailVerifiedEvent", event)
}

// PublishUserPhoneVerified 发布用户手机验证事件
func (p *OutboxPublisher) PublishUserPhoneVerified(event domain.UserPhoneVerifiedEvent) error {
	return p.publishEvent("UserPhoneVerifiedEvent", event)
}

// PublishUserLogin 发布用户登录事件
func (p *OutboxPublisher) PublishUserLogin(event domain.UserLoginEvent) error {
	return p.publishEvent("UserLoginEvent", event)
}

// PublishUserLogout 发布用户登出事件
func (p *OutboxPublisher) PublishUserLogout(event domain.UserLogoutEvent) error {
	return p.publishEvent("UserLogoutEvent", event)
}

// PublishUserFailedLogin 发布用户登录失败事件
func (p *OutboxPublisher) PublishUserFailedLogin(event domain.UserFailedLoginEvent) error {
	return p.publishEvent("UserFailedLoginEvent", event)
}

// PublishUserLocked 发布用户账户锁定事件
func (p *OutboxPublisher) PublishUserLocked(event domain.UserLockedEvent) error {
	return p.publishEvent("UserLockedEvent", event)
}

// PublishUserUnlocked 发布用户账户解锁事件
func (p *OutboxPublisher) PublishUserUnlocked(event domain.UserUnlockedEvent) error {
	return p.publishEvent("UserUnlockedEvent", event)
}

// publishEvent 通用发布事件方法
func (p *OutboxPublisher) publishEvent(eventType string, event interface{}) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	message := OutboxMessage{
		ID:        generateUUID(),
		EventID:   generateUUID(),
		EventType: eventType,
		Payload:   string(payload),
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return p.db.Create(&message).Error
}

// ProcessOutboxMessages 处理待处理的消息
func (p *OutboxPublisher) ProcessOutboxMessages(ctx context.Context, batchSize int) error {
	var messages []OutboxMessage

	// 查找待处理的消息
	if err := p.db.Where("status = ?", "pending").Limit(batchSize).Find(&messages).Error; err != nil {
		return err
	}

	for _, message := range messages {
		// 这里应该实现将消息发送到消息队列的逻辑
		// 例如使用 Kafka、RabbitMQ 等

		// 模拟发送成功
		if err := p.db.Model(&message).Update("status", "sent").Error; err != nil {
			return err
		}
	}

	return nil
}

// CleanupProcessedMessages 清理已处理的消息
func (p *OutboxPublisher) CleanupProcessedMessages(ctx context.Context, before time.Time) error {
	return p.db.Where("status = ? AND updated_at < ?", "sent", before).Delete(&OutboxMessage{}).Error
}

// generateUUID 生成 UUID
func generateUUID() string {
	return time.Now().Format("20060102150405") + "-" + time.Now().String()
}
