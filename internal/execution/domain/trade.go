package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/eventsourcing"
)

// Trade 成交单
type Trade struct {
	eventsourcing.AggregateRoot
	ID               uint            `json:"id"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	TradeID          string          `json:"trade_id"`
	OrderID          string          `json:"order_id"`
	UserID           string          `json:"user_id"`
	Symbol           string          `json:"symbol"`
	Side             TradeSide       `json:"side"`
	ExecutedPrice    decimal.Decimal `json:"executed_price"`
	ExecutedQuantity decimal.Decimal `json:"executed_quantity"`
	ExecutedAt       time.Time       `json:"executed_at"`
	Status           string          `json:"status"`
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
