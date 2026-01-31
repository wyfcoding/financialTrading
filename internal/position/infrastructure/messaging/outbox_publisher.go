package messaging

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/wyfcoding/financialtrading/internal/position/domain"
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
	return "position_outbox_messages"
}

// OutboxEventPublisher 实现 EventPublisher 接口，使用 Outbox 模式
type OutboxEventPublisher struct {
	db *gorm.DB
}

// NewOutboxEventPublisher 创建新的 OutboxEventPublisher 实例
func NewOutboxEventPublisher(db *gorm.DB) *OutboxEventPublisher {
	return &OutboxEventPublisher{db: db}
}

// PublishPositionCreated 发布头寸创建事件
func (p *OutboxEventPublisher) PublishPositionCreated(event domain.PositionCreatedEvent) error {
	return p.publishEvent("PositionCreatedEvent", event)
}

// PublishPositionUpdated 发布头寸更新事件
func (p *OutboxEventPublisher) PublishPositionUpdated(event domain.PositionUpdatedEvent) error {
	return p.publishEvent("PositionUpdatedEvent", event)
}

// PublishPositionClosed 发布头寸关闭事件
func (p *OutboxEventPublisher) PublishPositionClosed(event domain.PositionClosedEvent) error {
	return p.publishEvent("PositionClosedEvent", event)
}

// PublishPositionPnLUpdated 发布头寸盈亏更新事件
func (p *OutboxEventPublisher) PublishPositionPnLUpdated(event domain.PositionPnLUpdatedEvent) error {
	return p.publishEvent("PositionPnLUpdatedEvent", event)
}

// PublishPositionCostMethodChanged 发布头寸成本计算方法变更事件
func (p *OutboxEventPublisher) PublishPositionCostMethodChanged(event domain.PositionCostMethodChangedEvent) error {
	return p.publishEvent("PositionCostMethodChangedEvent", event)
}

// PublishPositionFlip 发布头寸反手事件
func (p *OutboxEventPublisher) PublishPositionFlip(event domain.PositionFlipEvent) error {
	return p.publishEvent("PositionFlipEvent", event)
}

// publishEvent 通用事件发布方法
func (p *OutboxEventPublisher) publishEvent(eventType string, event interface{}) error {
	// 序列化事件数据
	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// 创建 Outbox 记录
	message := OutboxMessage{
		ID:        generateUUID(),
		EventID:   generateUUID(),
		EventType: eventType,
		Payload:   string(eventData),
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 保存到数据库
	return p.db.Create(&message).Error
}

// ProcessOutboxMessages 处理待处理的消息
func (p *OutboxEventPublisher) ProcessOutboxMessages(ctx context.Context, batchSize int) error {
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
func (p *OutboxEventPublisher) CleanupProcessedMessages(ctx context.Context, before time.Time) error {
	return p.db.Where("status = ? AND updated_at < ?", "sent", before).Delete(&OutboxMessage{}).Error
}

// generateUUID 生成 UUID
func generateUUID() string {
	return time.Now().Format("20060102150405") + "-" + time.Now().String()
}
