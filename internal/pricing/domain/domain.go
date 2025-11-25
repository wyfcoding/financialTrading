package domain

import (
	"context"
	"time"
)

// OptionType 期权类型
type OptionType string

const (
	OptionTypeCall OptionType = "CALL"
	OptionTypePut  OptionType = "PUT"
)

// OptionContract 期权合约
type OptionContract struct {
	Symbol      string
	Type        OptionType
	StrikePrice float64
	ExpiryDate  time.Time
}

// Greeks 希腊字母
type Greeks struct {
	Delta float64
	Gamma float64
	Theta float64
	Vega  float64
	Rho   float64
}

// MarketDataClient 市场数据客户端接口
type MarketDataClient interface {
	GetPrice(ctx context.Context, symbol string) (float64, error)
}
