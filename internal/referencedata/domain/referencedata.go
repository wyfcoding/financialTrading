package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Symbol 交易对实体
// 领域层仅包含业务字段，不依赖具体存储实现。
type Symbol struct {
	ID             string          `json:"id"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	BaseCurrency   string          `json:"base_currency"`
	QuoteCurrency  string          `json:"quote_currency"`
	ExchangeID     string          `json:"exchange_id"`
	SymbolCode     string          `json:"symbol_code"`
	Status         string          `json:"status"`
	MinOrderSize   decimal.Decimal `json:"min_order_size"`
	PricePrecision decimal.Decimal `json:"price_precision"`
}

// Exchange 交易所实体
// 领域层仅包含业务字段，不依赖具体存储实现。
type Exchange struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
	Country   string    `json:"country"`
	Status    string    `json:"status"`
	Timezone  string    `json:"timezone"`
}
