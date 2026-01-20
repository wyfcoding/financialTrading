package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Trade 公共成交记录
type Trade struct {
	ID        string
	Symbol    string
	Price     decimal.Decimal
	Quantity  decimal.Decimal
	Side      string
	Timestamp time.Time
}
