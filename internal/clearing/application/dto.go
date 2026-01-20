package application

import "github.com/shopspring/decimal"

type SettleTradeRequest struct {
	TradeID    string
	BuyUserID  string
	SellUserID string
	Symbol     string
	Quantity   decimal.Decimal
	Price      decimal.Decimal
}

type SettleTradeCommand = SettleTradeRequest

type MarkSettlementCommand struct {
	SettlementID string
	Reason       string
	Success      bool
}

type SettlementDTO struct {
	SettlementID  string
	TradeID       string
	Status        string
	TotalAmount   string
	TradesSettled int64 // Added for GetClearingStatus
	TotalTrades   int64 // Added for GetClearingStatus
	SettledAt     int64
	ErrorMessage  string
}

type MarginDTO struct {
	Symbol           string
	BaseMarginRate   decimal.Decimal
	VolatilityFactor decimal.Decimal
}

func (m *MarginDTO) CurrentMarginRate() decimal.Decimal {
	return m.BaseMarginRate.Mul(m.VolatilityFactor)
}
