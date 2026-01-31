package messaging

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/catalog/domain"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"gorm.io/gorm"
)

// outboxPublisher 基于 Outbox 模式的事件发布者实现
type outboxPublisher struct {
	manager *outbox.Manager
}

// NewOutboxPublisher 创建一个新的 OutboxPublisher 实例
func NewOutboxPublisher(manager *outbox.Manager) domain.EventPublisher {
	return &outboxPublisher{manager: manager}
}

// Publish 发布一个普通事件（非事务内）
func (p *outboxPublisher) Publish(ctx context.Context, topic string, key string, event any) error {
	return p.manager.PublishInTx(ctx, p.manager.DB(), topic, key, event)
}

// PublishInTx 在事务中发布事件，核心用于 Outbox 模式
func (p *outboxPublisher) PublishInTx(ctx context.Context, tx any, topic string, key string, event any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok {
		return fmt.Errorf("tx must be *gorm.DB, got %T", tx)
	}
	return p.manager.PublishInTx(ctx, gormTx, topic, key, event)
}
