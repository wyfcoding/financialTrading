package application

import "github.com/shopspring/decimal"

type SettleTradeCommand struct {
	TradeID    string
	BuyUserID  string
	SellUserID string
	Symbol     string
	Quantity   decimal.Decimal
	Price      decimal.Decimal
}

type MarkSettlementCommand struct {
	SettlementID string
	Reason       string
	Success      bool
}

type SettlementDTO struct {
	SettlementID string
	TradeID      string
	Status       string
	TotalAmount  string
	SettledAt    int64
	ErrorMessage string
}
