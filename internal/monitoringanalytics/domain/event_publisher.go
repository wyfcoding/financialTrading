package domain

import "context"

// EventPublisher 统一事件发布接口（支持 Outbox）。
type EventPublisher interface {
	Publish(ctx context.Context, topic string, key string, event any) error
	PublishInTx(ctx context.Context, tx any, topic string, key string, event any) error
}
