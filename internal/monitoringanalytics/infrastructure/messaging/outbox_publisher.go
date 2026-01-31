package messaging

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
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
	return "monitoringanalytics_outbox_messages"
}

// OutboxEventPublisher 实现 EventPublisher 接口，使用 Outbox 模式
type OutboxEventPublisher struct {
	db *gorm.DB
}

// NewOutboxEventPublisher 创建新的 OutboxEventPublisher 实例
func NewOutboxEventPublisher(db *gorm.DB) *OutboxEventPublisher {
	return &OutboxEventPublisher{db: db}
}

// PublishMetricCreated 发布指标创建事件
func (p *OutboxEventPublisher) PublishMetricCreated(event domain.MetricCreatedEvent) error {
	return p.publishEvent("MetricCreatedEvent", event)
}

// PublishAlertGenerated 发布告警生成事件
func (p *OutboxEventPublisher) PublishAlertGenerated(event domain.AlertGeneratedEvent) error {
	return p.publishEvent("AlertGeneratedEvent", event)
}

// PublishAlertStatusChanged 发布告警状态变更事件
func (p *OutboxEventPublisher) PublishAlertStatusChanged(event domain.AlertStatusChangedEvent) error {
	return p.publishEvent("AlertStatusChangedEvent", event)
}

// PublishSystemHealthChanged 发布系统健康状态变更事件
func (p *OutboxEventPublisher) PublishSystemHealthChanged(event domain.SystemHealthChangedEvent) error {
	return p.publishEvent("SystemHealthChangedEvent", event)
}

// PublishExecutionAuditCreated 发布执行审计创建事件
func (p *OutboxEventPublisher) PublishExecutionAuditCreated(event domain.ExecutionAuditCreatedEvent) error {
	return p.publishEvent("ExecutionAuditCreatedEvent", event)
}

// PublishSpoofingDetected 发布哄骗检测事件
func (p *OutboxEventPublisher) PublishSpoofingDetected(event domain.SpoofingDetectedEvent) error {
	return p.publishEvent("SpoofingDetectedEvent", event)
}

// PublishMarketAnomalyDetected 发布市场异常检测事件
func (p *OutboxEventPublisher) PublishMarketAnomalyDetected(event domain.MarketAnomalyDetectedEvent) error {
	return p.publishEvent("MarketAnomalyDetectedEvent", event)
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
