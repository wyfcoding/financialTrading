package application

import "github.com/shopspring/decimal"

type IngestQuoteCommand struct {
	Symbol    string
	BidPrice  decimal.Decimal
	AskPrice  decimal.Decimal
	BidSize   decimal.Decimal
	AskSize   decimal.Decimal
	LastPrice decimal.Decimal
	LastSize  decimal.Decimal
	Source    string
}

type IngestTradeCommand struct {
	TradeID  string
	Symbol   string
	Price    decimal.Decimal
	Quantity decimal.Decimal
	Side     string
}

type KlineDTO struct {
	OpenTime  int64
	Open      string
	High      string
	Low       string
	Close     string
	Volume    string
	CloseTime int64
}

type TradeDTO struct {
	TradeID   string
	Symbol    string
	Price     string
	Quantity  string
	Side      string
	Timestamp int64
}
