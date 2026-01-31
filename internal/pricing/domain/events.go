package domain

import (
	"time"
)

// OptionPricedEvent 期权定价完成事件
type OptionPricedEvent struct {
	Symbol          string
	OptionType      OptionType
	StrikePrice     float64
	ExpiryDate      int64
	OptionPrice     float64
	UnderlyingPrice float64
	Volatility      float64
	RiskFreeRate    float64
	DividendYield   float64
	PricingModel    string
	CalculatedAt    int64
	OccurredOn      time.Time
}

// GreeksCalculatedEvent 希腊字母计算完成事件
type GreeksCalculatedEvent struct {
	Symbol          string
	OptionType      OptionType
	StrikePrice     float64
	ExpiryDate      int64
	UnderlyingPrice float64
	Delta           float64
	Gamma           float64
	Theta           float64
	Vega            float64
	Rho             float64
	CalculatedAt    int64
	OccurredOn      time.Time
}

// PricingModelChangedEvent 定价模型变更事件
type PricingModelChangedEvent struct {
	Symbol          string
	OldModel        string
	NewModel        string
	ChangedAt       int64
	OccurredOn      time.Time
}

// VolatilityUpdatedEvent 波动率更新事件
type VolatilityUpdatedEvent struct {
	Symbol          string
	OldVolatility   float64
	NewVolatility   float64
	UpdateReason    string
	UpdatedAt       int64
	OccurredOn      time.Time
}

// PricingErrorEvent 定价错误事件
type PricingErrorEvent struct {
	Symbol          string
	OptionType      OptionType
	StrikePrice     float64
	ExpiryDate      int64
	Error           string
	ErrorCode       string
	OccurredAt      int64
	OccurredOn      time.Time
}

// BatchPricingCompletedEvent 批量定价完成事件
type BatchPricingCompletedEvent struct {
	BatchID         string
	Symbols         []string
	TotalContracts  int
	SuccessCount    int
	FailureCount    int
	AverageTime     float64
	CompletedAt     int64
	OccurredOn      time.Time
}
