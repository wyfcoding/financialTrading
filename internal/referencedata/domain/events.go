package domain

import (
	"time"
)

// SymbolCreatedEvent 交易对创建事件
type SymbolCreatedEvent struct {
	SymbolID       string
	BaseCurrency   string
	QuoteCurrency  string
	ExchangeID     string
	SymbolCode     string
	Status         string
	MinOrderSize   float64
	PricePrecision float64
	CreatedAt      int64
	OccurredOn     time.Time
}

// SymbolUpdatedEvent 交易对更新事件
type SymbolUpdatedEvent struct {
	SymbolID        string
	OldStatus       string
	NewStatus       string
	OldMinOrderSize float64
	NewMinOrderSize float64
	UpdatedAt       int64
	OccurredOn      time.Time
}

// SymbolDeletedEvent 交易对删除事件
type SymbolDeletedEvent struct {
	SymbolID   string
	SymbolCode string
	DeletedAt  int64
	OccurredOn time.Time
}

// ExchangeCreatedEvent 交易所创建事件
type ExchangeCreatedEvent struct {
	ExchangeID string
	Name       string
	Country    string
	Status     string
	Timezone   string
	CreatedAt  int64
	OccurredOn time.Time
}

// ExchangeUpdatedEvent 交易所更新事件
type ExchangeUpdatedEvent struct {
	ExchangeID string
	OldStatus  string
	NewStatus  string
	OldCountry string
	NewCountry string
	UpdatedAt  int64
	OccurredOn time.Time
}

// ExchangeDeletedEvent 交易所删除事件
type ExchangeDeletedEvent struct {
	ExchangeID string
	Name       string
	DeletedAt  int64
	OccurredOn time.Time
}

// SymbolStatusChangedEvent 交易对状态变更事件
type SymbolStatusChangedEvent struct {
	SymbolID   string
	SymbolCode string
	OldStatus  string
	NewStatus  string
	ChangedAt  int64
	OccurredOn time.Time
}

// ExchangeStatusChangedEvent 交易所状态变更事件
type ExchangeStatusChangedEvent struct {
	ExchangeID string
	Name       string
	OldStatus  string
	NewStatus  string
	ChangedAt  int64
	OccurredOn time.Time
}
