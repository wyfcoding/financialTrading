package domain

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type OrderSide string

const (
	SideBuy  OrderSide = "BUY"
	SideSell OrderSide = "SELL"
)

type OrderStatus string

const (
	StatusNew             OrderStatus = "NEW"
	StatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	StatusFilled          OrderStatus = "FILLED"
	StatusCancelled       OrderStatus = "CANCELLED"
)

// DarkOrder 暗池订单实体
type DarkOrder struct {
	gorm.Model
	OrderID        string          `gorm:"column:order_id;type:varchar(32);unique_index;not null"`
	UserID         string          `gorm:"column:user_id;type:varchar(32);index;not null"`
	Symbol         string          `gorm:"column:symbol;type:varchar(20);not null"`
	Side           OrderSide       `gorm:"column:side;type:varchar(10);not null"`
	Price          decimal.Decimal `gorm:"column:price;type:decimal(32,16);not null"`
	Quantity       decimal.Decimal `gorm:"column:quantity;type:decimal(32,16);not null"`
	FilledQuantity decimal.Decimal `gorm:"column:filled_quantity;type:decimal(32,16);default:0"`
	Status         OrderStatus     `gorm:"column:status;type:varchar(20);not null;default:'NEW'"`
}

func (DarkOrder) TableName() string { return "dark_orders" }
