package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Trade 成交单
type Trade struct {
	gorm.Model
	TradeID          string          `gorm:"column:trade_id;type:varchar(32);uniqueIndex;not null;comment:成交ID"`
	OrderID          string          `gorm:"column:order_id;type:varchar(32);index;not null;comment:订单ID"`
	UserID           string          `gorm:"column:user_id;type:varchar(32);index;not null;comment:用户ID"`
	Symbol           string          `gorm:"column:symbol;type:varchar(20);not null;comment:标的"`
	Side             TradeSide       `gorm:"column:side;type:varchar(10);not null;comment:方向"`
	ExecutedPrice    decimal.Decimal `gorm:"column:price;type:decimal(32,18);not null;comment:成交价"`
	ExecutedQuantity decimal.Decimal `gorm:"column:quantity;type:decimal(32,18);not null;comment:成交量"`
	ExecutedAt       time.Time       `gorm:"column:executed_at;not null;comment:成交时间"`
}

func (Trade) TableName() string {
	return "trades"
}

func NewTrade(tradeID, orderID, userID, symbol string, side TradeSide, price, qty decimal.Decimal) *Trade {
	return &Trade{
		TradeID:          tradeID,
		OrderID:          orderID,
		UserID:           userID,
		Symbol:           symbol,
		Side:             side,
		ExecutedPrice:    price,
		ExecutedQuantity: qty,
		ExecutedAt:       time.Now(),
	}
}
