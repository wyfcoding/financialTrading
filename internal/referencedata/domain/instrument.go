package domain

import (
	"gorm.io/gorm"
)

type InstrumentType string

const (
	Spot   InstrumentType = "SPOT"
	Future InstrumentType = "FUTURE"
	Option InstrumentType = "OPTION"
)

type Instrument struct {
	gorm.Model
	Symbol        string         `gorm:"column:symbol;type:varchar(20);uniqueIndex;not null"`
	BaseCurrency  string         `gorm:"column:base_currency;type:varchar(10);not null"`
	QuoteCurrency string         `gorm:"column:quote_currency;type:varchar(10);not null"`
	TickSize      float64        `gorm:"column:tick_size;type:decimal(20,8);not null"`
	LotSize       float64        `gorm:"column:lot_size;type:decimal(20,8);not null"`
	Type          InstrumentType `gorm:"column:type;type:varchar(10);not null"`
	MaxLeverage   int            `gorm:"column:max_leverage;default:1"`
}

func (Instrument) TableName() string {
	return "instruments"
}

func NewInstrument(symbol, base, quote string, tick, lot float64, typ InstrumentType) *Instrument {
	return &Instrument{
		Symbol:        symbol,
		BaseCurrency:  base,
		QuoteCurrency: quote,
		TickSize:      tick,
		LotSize:       lot,
		Type:          typ,
		MaxLeverage:   1, // default
	}
}
