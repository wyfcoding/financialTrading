package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Trade 成交单聚合物
type Trade struct {
	ID               string
	OrderID          string
	UserID           string
	Symbol           string
	Side             TradeSide
	ExecutedPrice    decimal.Decimal
	ExecutedQuantity decimal.Decimal
	ExecutedAt       time.Time
	CreatedAt        time.Time
}

func NewTrade(id, orderID, userID, symbol string, side TradeSide, price, qty decimal.Decimal) *Trade {
	return &Trade{
		ID:               id,
		OrderID:          orderID,
		UserID:           userID,
		Symbol:           symbol,
		Side:             side,
		ExecutedPrice:    price,
		ExecutedQuantity: qty,
		ExecutedAt:       time.Now(),
		CreatedAt:        time.Now(),
	}
}
