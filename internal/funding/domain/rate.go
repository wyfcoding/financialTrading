package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// FundingRate 资金费率记录
type FundingRate struct {
	gorm.Model
	Symbol    string          `gorm:"column:symbol;type:varchar(20);index;not null"`
	Rate      decimal.Decimal `gorm:"column:rate;type:decimal(16,8);not null"`
	Timestamp time.Time       `gorm:"column:timestamp;not null"`
}

func (FundingRate) TableName() string { return "funding_rates" }
