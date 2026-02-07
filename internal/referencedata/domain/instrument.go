package domain

import "time"

type InstrumentType string

const (
	Spot   InstrumentType = "SPOT"
	Future InstrumentType = "FUTURE"
	Option InstrumentType = "OPTION"
)

// Instrument 合约/交易品种
// 领域层仅包含业务字段，不依赖具体存储实现。
type Instrument struct {
	ID            string         `json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	Symbol        string         `json:"symbol"`
	BaseCurrency  string         `json:"base_currency"`
	QuoteCurrency string         `json:"quote_currency"`
	TickSize      float64        `json:"tick_size"`
	LotSize       float64        `json:"lot_size"`
	Type          InstrumentType `json:"type"`
	MaxLeverage   int            `json:"max_leverage"`
}

func NewInstrument(symbol, base, quote string, tick, lot float64, typ InstrumentType) *Instrument {
	return &Instrument{
		Symbol:        symbol,
		BaseCurrency:  base,
		QuoteCurrency: quote,
		TickSize:      tick,
		LotSize:       lot,
		Type:          typ,
		MaxLeverage:   1,
	}
}
