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

// Update 根据新成交价格更新 K 线
func (k *Kline) Update(price, qty decimal.Decimal) {
	if price.GreaterThan(k.High) {
		k.High = price
	}
	if price.LessThan(k.Low) {
		k.Low = price
	}
	k.Close = price
	k.Volume = k.Volume.Add(qty)
}
