package domain

import "time"

const (
	SymbolCreatedEventType       = "referencedata.symbol.created"
	SymbolUpdatedEventType       = "referencedata.symbol.updated"
	SymbolDeletedEventType       = "referencedata.symbol.deleted"
	SymbolStatusChangedEventType = "referencedata.symbol.status_changed"
	ExchangeCreatedEventType     = "referencedata.exchange.created"
	ExchangeUpdatedEventType     = "referencedata.exchange.updated"
	ExchangeDeletedEventType     = "referencedata.exchange.deleted"
	ExchangeStatusChangedEventType = "referencedata.exchange.status_changed"
)

// SymbolCreatedEvent 交易对创建事件
type SymbolCreatedEvent struct {
	SymbolID       string    `json:"symbol_id"`
	BaseCurrency   string    `json:"base_currency"`
	QuoteCurrency  string    `json:"quote_currency"`
	ExchangeID     string    `json:"exchange_id"`
	SymbolCode     string    `json:"symbol_code"`
	Status         string    `json:"status"`
	MinOrderSize   float64   `json:"min_order_size"`
	PricePrecision float64   `json:"price_precision"`
	CreatedAt      int64     `json:"created_at"`
	OccurredOn     time.Time `json:"occurred_on"`
}

// SymbolUpdatedEvent 交易对更新事件
type SymbolUpdatedEvent struct {
	SymbolID        string    `json:"symbol_id"`
	OldStatus       string    `json:"old_status"`
	NewStatus       string    `json:"new_status"`
	OldMinOrderSize float64   `json:"old_min_order_size"`
	NewMinOrderSize float64   `json:"new_min_order_size"`
	UpdatedAt       int64     `json:"updated_at"`
	OccurredOn      time.Time `json:"occurred_on"`
}

// SymbolDeletedEvent 交易对删除事件
type SymbolDeletedEvent struct {
	SymbolID   string    `json:"symbol_id"`
	SymbolCode string    `json:"symbol_code"`
	DeletedAt  int64     `json:"deleted_at"`
	OccurredOn time.Time `json:"occurred_on"`
}

// ExchangeCreatedEvent 交易所创建事件
type ExchangeCreatedEvent struct {
	ExchangeID string    `json:"exchange_id"`
	Name       string    `json:"name"`
	Country    string    `json:"country"`
	Status     string    `json:"status"`
	Timezone   string    `json:"timezone"`
	CreatedAt  int64     `json:"created_at"`
	OccurredOn time.Time `json:"occurred_on"`
}

// ExchangeUpdatedEvent 交易所更新事件
type ExchangeUpdatedEvent struct {
	ExchangeID string    `json:"exchange_id"`
	OldStatus  string    `json:"old_status"`
	NewStatus  string    `json:"new_status"`
	OldCountry string    `json:"old_country"`
	NewCountry string    `json:"new_country"`
	UpdatedAt  int64     `json:"updated_at"`
	OccurredOn time.Time `json:"occurred_on"`
}

// ExchangeDeletedEvent 交易所删除事件
type ExchangeDeletedEvent struct {
	ExchangeID string    `json:"exchange_id"`
	Name       string    `json:"name"`
	DeletedAt  int64     `json:"deleted_at"`
	OccurredOn time.Time `json:"occurred_on"`
}

// SymbolStatusChangedEvent 交易对状态变更事件
type SymbolStatusChangedEvent struct {
	SymbolID   string    `json:"symbol_id"`
	SymbolCode string    `json:"symbol_code"`
	OldStatus  string    `json:"old_status"`
	NewStatus  string    `json:"new_status"`
	ChangedAt  int64     `json:"changed_at"`
	OccurredOn time.Time `json:"occurred_on"`
}

// ExchangeStatusChangedEvent 交易所状态变更事件
type ExchangeStatusChangedEvent struct {
	ExchangeID string    `json:"exchange_id"`
	Name       string    `json:"name"`
	OldStatus  string    `json:"old_status"`
	NewStatus  string    `json:"new_status"`
	ChangedAt  int64     `json:"changed_at"`
	OccurredOn time.Time `json:"occurred_on"`
}
