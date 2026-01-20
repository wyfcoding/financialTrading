package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Kline K线数据
type Kline struct {
	Symbol    string
	Interval  string
	OpenTime  time.Time
	CloseTime time.Time
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Volume    decimal.Decimal
}

func NewKline(symbol, interval string, openTime, closeTime time.Time, o, h, l, c, v decimal.Decimal) *Kline {
	return &Kline{
		Symbol:    symbol,
		Interval:  interval,
		OpenTime:  openTime,
		CloseTime: closeTime,
		Open:      o,
		High:      h,
		Low:       l,
		Close:     c,
		Volume:    v,
	}
}
