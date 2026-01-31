package messaging

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
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
	return "pricing_outbox_messages"
}

// OutboxEventPublisher 实现 EventPublisher 接口，使用 Outbox 模式
type OutboxEventPublisher struct {
	db *gorm.DB
}

// NewOutboxEventPublisher 创建新的 OutboxEventPublisher 实例
func NewOutboxEventPublisher(db *gorm.DB) *OutboxEventPublisher {
	return &OutboxEventPublisher{db: db}
}

// PublishOptionPriced 发布期权定价完成事件
func (p *OutboxEventPublisher) PublishOptionPriced(event domain.OptionPricedEvent) error {
	return p.publishEvent("OptionPricedEvent", event)
}

// PublishGreeksCalculated 发布希腊字母计算完成事件
func (p *OutboxEventPublisher) PublishGreeksCalculated(event domain.GreeksCalculatedEvent) error {
	return p.publishEvent("GreeksCalculatedEvent", event)
}

// PublishPricingModelChanged 发布定价模型变更事件
func (p *OutboxEventPublisher) PublishPricingModelChanged(event domain.PricingModelChangedEvent) error {
	return p.publishEvent("PricingModelChangedEvent", event)
}

// PublishVolatilityUpdated 发布波动率更新事件
func (p *OutboxEventPublisher) PublishVolatilityUpdated(event domain.VolatilityUpdatedEvent) error {
	return p.publishEvent("VolatilityUpdatedEvent", event)
}

// PublishPricingError 发布定价错误事件
func (p *OutboxEventPublisher) PublishPricingError(event domain.PricingErrorEvent) error {
	return p.publishEvent("PricingErrorEvent", event)
}

// PublishBatchPricingCompleted 发布批量定价完成事件
func (p *OutboxEventPublisher) PublishBatchPricingCompleted(event domain.BatchPricingCompletedEvent) error {
	return p.publishEvent("BatchPricingCompletedEvent", event)
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
