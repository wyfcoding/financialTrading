package domain

import (
	"time"

	"github.com/shopspring/decimal"
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
	Symbol      string          `json:"symbol"`
	Type        OptionType      `json:"type"`
	StrikePrice decimal.Decimal `json:"strike_price"`
	ExpiryDate  int64           `json:"expiry_date"`
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
	ID              uint            `json:"id"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Symbol          string          `json:"symbol"`
	OptionPrice     decimal.Decimal `json:"option_price"`
	UnderlyingPrice decimal.Decimal `json:"underlying_price"`
	Delta           decimal.Decimal `json:"delta"`
	Gamma           decimal.Decimal `json:"gamma"`
	Theta           decimal.Decimal `json:"theta"`
	Vega            decimal.Decimal `json:"vega"`
	Rho             decimal.Decimal `json:"rho"`
	CalculatedAt    int64           `json:"calculated_at"`
	PricingModel    string          `json:"pricing_model"`
}
