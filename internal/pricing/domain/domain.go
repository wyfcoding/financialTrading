// 包 定价服务的领域模型
package domain

import (
	"context"
	"time"
)

// OptionType 期权类型
type OptionType string

const (
	OptionTypeCall OptionType = "CALL" // 看涨期权
	OptionTypePut  OptionType = "PUT"  // 看跌期权
)

// OptionContract 期权合约
// 定义期权的基本属性
type OptionContract struct {
	Symbol      string     // 标的资产代码
	Type        OptionType // 期权类型 (CALL/PUT)
	StrikePrice float64    // 行权价
	ExpiryDate  time.Time  // 到期日
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
