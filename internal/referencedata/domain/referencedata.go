// 包 参考数据服务的领域模型
package domain

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Symbol 交易对实体
type Symbol struct {
	gorm.Model
	ID             string          `gorm:"column:id;type:varchar(32);primaryKey" json:"id"`
	BaseCurrency   string          `gorm:"column:base_currency;type:varchar(10);not null" json:"base_currency"`
	QuoteCurrency  string          `gorm:"column:quote_currency;type:varchar(10);not null" json:"quote_currency"`
	ExchangeID     string          `gorm:"column:exchange_id;type:varchar(32);not null;index" json:"exchange_id"`
	SymbolCode     string          `gorm:"column:symbol_code;type:varchar(20);uniqueIndex;not null" json:"symbol_code"`
	Status         string          `gorm:"column:status;type:varchar(20);default:'ACTIVE'" json:"status"`
	MinOrderSize   decimal.Decimal `gorm:"column:min_order_size;type:decimal(20,8);default:0" json:"min_order_size"`
	PricePrecision decimal.Decimal `gorm:"column:price_precision;type:decimal(20,8);default:0" json:"price_precision"`
}

// Exchange 交易所实体
type Exchange struct {
	gorm.Model
	ID       string `gorm:"column:id;type:varchar(32);primaryKey" json:"id"`
	Name     string `gorm:"column:name;type:varchar(50);uniqueIndex;not null" json:"name"`
	Country  string `gorm:"column:country;type:varchar(50)" json:"country"`
	Status   string `gorm:"column:status;type:varchar(20);default:'ACTIVE'" json:"status"`
	Timezone string `gorm:"column:timezone;type:varchar(50)" json:"timezone"`
}

// End of domain file
