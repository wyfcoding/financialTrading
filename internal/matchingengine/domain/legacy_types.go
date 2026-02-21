package domain

import "github.com/shopspring/decimal"

type OrderType int8

const (
	Limit  OrderType = 1
	Market OrderType = 2
)

type OrderStatus int8

const (
	New             OrderStatus = 1
	PartiallyFilled OrderStatus = 2
	Filled          OrderStatus = 3
	Canceled        OrderStatus = 4
)

// Order is a compatibility shape used by legacy in-memory matching adapters.
type Order struct {
	ID        uint64
	Symbol    string
	Side      OrderSide
	Type      OrderType
	Price     decimal.Decimal
	Quantity  decimal.Decimal
	FilledQty decimal.Decimal
	Timestamp int64
	Status    OrderStatus
}

// OrderBookLevel keeps backward compatibility with older persistence mappings.
type OrderBookLevel = PriceLevel
