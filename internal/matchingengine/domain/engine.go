package domain

import (
	"github.com/shopspring/decimal"
)

type PriceLevel struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
}
