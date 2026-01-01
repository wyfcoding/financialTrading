// 包 定价服务的领域模型
package domain

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// OptionType 期权类型
type OptionType string

const (
	OptionTypeCall OptionType = "CALL" // 看涨期权
	OptionTypePut  OptionType = "PUT"  // 看跌期权
)

// OptionContract 期权合约
// 定义期权的基本属性
type OptionContract struct {
	gorm.Model
	Symbol      string          `gorm:"column:symbol;type:varchar(32);index;not null"`
	Type        OptionType      `gorm:"column:type;type:varchar(10);not null"`
	StrikePrice decimal.Decimal `gorm:"column:strike_price;type:decimal(32,18);not null"`
	ExpiryDate  int64           `gorm:"column:expiry_date;type:bigint;not null"`
}

// Greeks 希腊字母
type Greeks struct {
	Delta decimal.Decimal
	Gamma decimal.Decimal
	Theta decimal.Decimal
	Vega  decimal.Decimal
	Rho   decimal.Decimal
}

// PricingResult 定价结果实体
type PricingResult struct {
	gorm.Model
	Symbol          string          `gorm:"column:symbol;type:varchar(32);index;not null"`
	OptionPrice     decimal.Decimal `gorm:"column:option_price;type:decimal(32,18);not null"`
	UnderlyingPrice decimal.Decimal `gorm:"column:underlying_price;type:decimal(32,18);not null"`
	Delta           decimal.Decimal `gorm:"column:delta;type:decimal(32,18)"`
	Gamma           decimal.Decimal `gorm:"column:gamma;type:decimal(32,18)"`
	Theta           decimal.Decimal `gorm:"column:theta;type:decimal(32,18)"`
	Vega            decimal.Decimal `gorm:"column:vega;type:decimal(32,18)"`
	Rho             decimal.Decimal `gorm:"column:rho;type:decimal(32,18)"`
	CalculatedAt    int64           `gorm:"column:calculated_at;type:bigint;not null"`
}

// End of domain file
