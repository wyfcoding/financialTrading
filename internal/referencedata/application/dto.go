package application

type InstrumentDTO struct {
	Symbol        string  `json:"symbol"`
	BaseCurrency  string  `json:"base_currency"`
	QuoteCurrency string  `json:"quote_currency"`
	TickSize      float64 `json:"tick_size"`
	LotSize       float64 `json:"lot_size"`
	Type          string  `json:"type"`
	MaxLeverage   int     `json:"max_leverage"`
}
