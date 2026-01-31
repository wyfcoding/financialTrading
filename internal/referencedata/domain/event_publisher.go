package domain

// EventPublisher 事件发布者接口
type EventPublisher interface {
	// PublishSymbolCreated 发布交易对创建事件
	PublishSymbolCreated(event SymbolCreatedEvent) error

	// PublishSymbolUpdated 发布交易对更新事件
	PublishSymbolUpdated(event SymbolUpdatedEvent) error

	// PublishSymbolDeleted 发布交易对删除事件
	PublishSymbolDeleted(event SymbolDeletedEvent) error

	// PublishExchangeCreated 发布交易所创建事件
	PublishExchangeCreated(event ExchangeCreatedEvent) error

	// PublishExchangeUpdated 发布交易所更新事件
	PublishExchangeUpdated(event ExchangeUpdatedEvent) error

	// PublishExchangeDeleted 发布交易所删除事件
	PublishExchangeDeleted(event ExchangeDeletedEvent) error

	// PublishSymbolStatusChanged 发布交易对状态变更事件
	PublishSymbolStatusChanged(event SymbolStatusChangedEvent) error

	// PublishExchangeStatusChanged 发布交易所状态变更事件
	PublishExchangeStatusChanged(event ExchangeStatusChangedEvent) error
}
