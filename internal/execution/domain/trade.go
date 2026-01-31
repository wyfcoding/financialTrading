package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/eventsourcing"
	"gorm.io/gorm"
)

// Trade 成交单
type Trade struct {
	gorm.Model
	eventsourcing.AggregateRoot
	TradeID          string          `gorm:"column:trade_id;type:varchar(32);uniqueIndex;not null;comment:成交ID"`
	OrderID          string          `gorm:"column:order_id;type:varchar(32);index;not null;comment:订单ID"`
	UserID           string          `gorm:"column:user_id;type:varchar(32);index;not null;comment:用户ID"`
	Symbol           string          `gorm:"column:symbol;type:varchar(20);not null;comment:标的"`
	Side             TradeSide       `gorm:"column:side;type:varchar(10);not null;comment:方向"`
	ExecutedPrice    decimal.Decimal `gorm:"column:price;type:decimal(32,18);not null;comment:成交价"`
	ExecutedQuantity decimal.Decimal `gorm:"column:quantity;type:decimal(32,18);not null;comment:成交量"`
	ExecutedAt       time.Time       `gorm:"column:executed_at;not null;comment:成交时间"`
	Status           string          `gorm:"column:status;type:varchar(20);default:'EXECUTED';comment:状态"`
}

func (Trade) TableName() string {
	return "trades"
}

func NewTrade(tradeID, orderID, userID, symbol string, side TradeSide, price, qty decimal.Decimal) *Trade {
	t := &Trade{
		TradeID:          tradeID,
		OrderID:          orderID,
		UserID:           userID,
		Symbol:           symbol,
		Side:             side,
		ExecutedPrice:    price,
		ExecutedQuantity: qty,
		ExecutedAt:       time.Now(),
		Status:           "EXECUTED",
	}
	t.SetID(tradeID)

	t.ApplyChange(&TradeExecutedEvent{
		TradeID:  tradeID,
		OrderID:  orderID,
		UserID:   userID,
		Symbol:   symbol,
		Quantity: qty.String(),
		Price:    price.String(),
		Time:     t.ExecutedAt.Unix(),
	})
	return t
}

func (t *Trade) Apply(event eventsourcing.DomainEvent) {
	switch e := event.(type) {
	case *TradeExecutedEvent:
		t.TradeID = e.TradeID
		t.OrderID = e.OrderID
		t.UserID = e.UserID
		t.Symbol = e.Symbol
		t.ExecutedQuantity, _ = decimal.NewFromString(e.Quantity)
		t.ExecutedPrice, _ = decimal.NewFromString(e.Price)
		t.Status = "EXECUTED"
	}
}
