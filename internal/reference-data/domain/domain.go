// 包 参考数据服务的领域模型
package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Symbol 交易对实体
// 定义交易对的基本属性
type Symbol struct {
	gorm.Model
	// 交易对 ID，UUID
	ID string `gorm:"column:id;type:varchar(36);primaryKey" json:"id"`
	// 基础货币 (例如 BTC)
	BaseCurrency string `gorm:"column:base_currency;type:varchar(10);not null" json:"base_currency"`
	// 计价货币 (例如 USD)
	QuoteCurrency string `gorm:"column:quote_currency;type:varchar(10);not null" json:"quote_currency"`
	// 所属交易所 ID
	ExchangeID string `gorm:"column:exchange_id;type:varchar(36);not null;index" json:"exchange_id"`
	// 交易对代码 (例如 BTC/USD)
	SymbolCode string `gorm:"column:symbol_code;type:varchar(20);uniqueIndex;not null" json:"symbol_code"`
	// 状态 (ACTIVE/INACTIVE)
	Status string `gorm:"column:status;type:varchar(20);default:'ACTIVE'" json:"status"`
	// 最小下单数量
	MinOrderSize float64 `gorm:"column:min_order_size;type:decimal(20,8);default:0" json:"min_order_size"`
	// 价格精度
	PricePrecision float64 `gorm:"column:price_precision;type:decimal(20,8);default:0" json:"price_precision"`
	// 创建时间
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
	// 更新时间
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime" json:"updated_at"`
}

// Exchange 交易所实体
type Exchange struct {
	gorm.Model
	ID        string    `gorm:"column:id;type:varchar(36);primaryKey" json:"id"`
	Name      string    `gorm:"column:name;type:varchar(50);uniqueIndex;not null" json:"name"`
	Country   string    `gorm:"column:country;type:varchar(50)" json:"country"`
	Status    string    `gorm:"column:status;type:varchar(20);default:'ACTIVE'" json:"status"`
	Timezone  string    `gorm:"column:timezone;type:varchar(50)" json:"timezone"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime" json:"updated_at"`
}

// SymbolRepository 交易对仓储接口
type SymbolRepository interface {
	GetByID(ctx context.Context, id string) (*Symbol, error)
	GetByCode(ctx context.Context, code string) (*Symbol, error)
	List(ctx context.Context, exchangeID string, status string, limit int, offset int) ([]*Symbol, error)
}

// ExchangeRepository 交易所仓储接口
type ExchangeRepository interface {
	GetByID(ctx context.Context, id string) (*Exchange, error)
	List(ctx context.Context, limit int, offset int) ([]*Exchange, error)
}
