package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

type FIXOrderCommand struct {
	ClOrdID  string
	UserID   string
	Symbol   string
	Side     string
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

type ExecutionClient interface {
	SubmitOrder(ctx context.Context, cmd FIXOrderCommand) (string, error)
}
