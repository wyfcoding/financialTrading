package messaging

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
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
	return "risk_outbox_messages"
}

// OutboxEventPublisher 实现 EventPublisher 接口，使用 Outbox 模式
type OutboxEventPublisher struct {
	db *gorm.DB
}

// NewOutboxEventPublisher 创建新的 OutboxEventPublisher 实例
func NewOutboxEventPublisher(db *gorm.DB) *OutboxEventPublisher {
	return &OutboxEventPublisher{db: db}
}

// PublishRiskAssessmentCreated 发布风险评估创建事件
func (p *OutboxEventPublisher) PublishRiskAssessmentCreated(event domain.RiskAssessmentCreatedEvent) error {
	return p.publishEvent("RiskAssessmentCreatedEvent", event)
}

// PublishRiskLimitExceeded 发布风险限额超出事件
func (p *OutboxEventPublisher) PublishRiskLimitExceeded(event domain.RiskLimitExceededEvent) error {
	return p.publishEvent("RiskLimitExceededEvent", event)
}

// PublishCircuitBreakerFired 发布熔断触发事件
func (p *OutboxEventPublisher) PublishCircuitBreakerFired(event domain.CircuitBreakerFiredEvent) error {
	return p.publishEvent("CircuitBreakerFiredEvent", event)
}

// PublishCircuitBreakerReset 发布熔断重置事件
func (p *OutboxEventPublisher) PublishCircuitBreakerReset(event domain.CircuitBreakerResetEvent) error {
	return p.publishEvent("CircuitBreakerResetEvent", event)
}

// PublishRiskAlertGenerated 发布风险告警生成事件
func (p *OutboxEventPublisher) PublishRiskAlertGenerated(event domain.RiskAlertGeneratedEvent) error {
	return p.publishEvent("RiskAlertGeneratedEvent", event)
}

// PublishMarginCall 发布追加保证金通知事件
func (p *OutboxEventPublisher) PublishMarginCall(event domain.MarginCallEvent) error {
	return p.publishEvent("MarginCallEvent", event)
}

// PublishRiskMetricsUpdated 发布风险指标更新事件
func (p *OutboxEventPublisher) PublishRiskMetricsUpdated(event domain.RiskMetricsUpdatedEvent) error {
	return p.publishEvent("RiskMetricsUpdatedEvent", event)
}

// PublishRiskLevelChanged 发布风险等级变更事件
func (p *OutboxEventPublisher) PublishRiskLevelChanged(event domain.RiskLevelChangedEvent) error {
	return p.publishEvent("RiskLevelChangedEvent", event)
}

// PublishPositionLiquidationTriggered 发布强平触发事件
func (p *OutboxEventPublisher) PublishPositionLiquidationTriggered(event domain.PositionLiquidationTriggeredEvent) error {
	return p.publishEvent("PositionLiquidationTriggeredEvent", event)
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
