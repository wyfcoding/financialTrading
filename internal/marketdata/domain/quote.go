package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Quote 实时报价聚合根
type Quote struct {
	Symbol    string
	BidPrice  decimal.Decimal
	AskPrice  decimal.Decimal
	BidSize   decimal.Decimal
	AskSize   decimal.Decimal
	LastPrice decimal.Decimal
	LastSize  decimal.Decimal
	Timestamp time.Time
}

func NewQuote(symbol string, bidPx, askPx, bidSz, askSz, lastPx, lastSz decimal.Decimal) *Quote {
	return &Quote{
		Symbol:    symbol,
		BidPrice:  bidPx,
		AskPrice:  askPx,
		BidSize:   bidSz,
		AskSize:   askSz,
		LastPrice: lastPx,
		LastSize:  lastSz,
		Timestamp: time.Now(),
	}
}
