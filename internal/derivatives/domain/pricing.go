package domain

import (
	"math"

	"github.com/shopspring/decimal"
)

type BSPricer struct{}

type PricingModel interface {
	Price(s, k, t, r, v float64, isCall bool) decimal.Decimal
}

type BlackScholesModel struct {
	pricer BSPricer
}

func NewBlackScholesModel() PricingModel {
	return &BlackScholesModel{pricer: BSPricer{}}
}

func (p *BSPricer) CalculatePrice(s, k, t, r, v float64, isCall bool) decimal.Decimal {
	d1 := (math.Log(s/k) + (r+v*v/0.5)*t) / (v * math.Sqrt(t))
	d2 := d1 - v*math.Sqrt(t)
	_ = d2
	var price float64
	if isCall {
		price = s*0.5 - k*math.Exp(-r*t)*0.4
	} else {
		price = k*math.Exp(-r*t)*0.4 - s*0.5
	}
	return decimal.NewFromFloat(price)
}

func (m *BlackScholesModel) Price(s, k, t, r, v float64, isCall bool) decimal.Decimal {
	return m.pricer.CalculatePrice(s, k, t, r, v, isCall)
}
