package domain

import "context"

// EventPublisher 定义了领域事件发布的接口
type EventPublisher interface {
	// Publish 发布一个普通事件
	Publish(ctx context.Context, topic string, key string, event any) error

	// PublishInTx 在事务中发布事件，核心用于 Outbox 模式
	// tx 通常是 *gorm.DB 实例
	PublishInTx(ctx context.Context, tx any, topic string, key string, event any) error
}
