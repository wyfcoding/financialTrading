package domain

import "time"

const (
	OptionPricedEventType          = "OptionPriced"
	GreeksCalculatedEventType      = "GreeksCalculated"
	PricingModelChangedEventType   = "PricingModelChanged"
	VolatilityUpdatedEventType     = "VolatilityUpdated"
	PricingErrorEventType          = "PricingError"
	BatchPricingCompletedEventType = "BatchPricingCompleted"
)

// OptionPricedEvent 期权定价完成事件
type OptionPricedEvent struct {
	Symbol          string     `json:"symbol"`
	OptionType      OptionType `json:"option_type"`
	StrikePrice     float64    `json:"strike_price"`
	ExpiryDate      int64      `json:"expiry_date"`
	OptionPrice     float64    `json:"option_price"`
	UnderlyingPrice float64    `json:"underlying_price"`
	Volatility      float64    `json:"volatility"`
	RiskFreeRate    float64    `json:"risk_free_rate"`
	DividendYield   float64    `json:"dividend_yield"`
	PricingModel    string     `json:"pricing_model"`
	CalculatedAt    int64      `json:"calculated_at"`
	OccurredOn      time.Time  `json:"occurred_on"`
}

// GreeksCalculatedEvent 希腊字母计算完成事件
type GreeksCalculatedEvent struct {
	Symbol          string     `json:"symbol"`
	OptionType      OptionType `json:"option_type"`
	StrikePrice     float64    `json:"strike_price"`
	ExpiryDate      int64      `json:"expiry_date"`
	UnderlyingPrice float64    `json:"underlying_price"`
	Delta           float64    `json:"delta"`
	Gamma           float64    `json:"gamma"`
	Theta           float64    `json:"theta"`
	Vega            float64    `json:"vega"`
	Rho             float64    `json:"rho"`
	CalculatedAt    int64      `json:"calculated_at"`
	OccurredOn      time.Time  `json:"occurred_on"`
}

// PricingModelChangedEvent 定价模型变更事件
type PricingModelChangedEvent struct {
	Symbol     string    `json:"symbol"`
	OldModel   string    `json:"old_model"`
	NewModel   string    `json:"new_model"`
	ChangedAt  int64     `json:"changed_at"`
	OccurredOn time.Time `json:"occurred_on"`
}

// VolatilityUpdatedEvent 波动率更新事件
type VolatilityUpdatedEvent struct {
	Symbol        string    `json:"symbol"`
	OldVolatility float64   `json:"old_volatility"`
	NewVolatility float64   `json:"new_volatility"`
	UpdateReason  string    `json:"update_reason"`
	UpdatedAt     int64     `json:"updated_at"`
	OccurredOn    time.Time `json:"occurred_on"`
}

// PricingErrorEvent 定价错误事件
type PricingErrorEvent struct {
	Symbol      string     `json:"symbol"`
	OptionType  OptionType `json:"option_type"`
	StrikePrice float64    `json:"strike_price"`
	ExpiryDate  int64      `json:"expiry_date"`
	Error       string     `json:"error"`
	ErrorCode   string     `json:"error_code"`
	OccurredAt  int64      `json:"occurred_at"`
	OccurredOn  time.Time  `json:"occurred_on"`
}

// BatchPricingCompletedEvent 批量定价完成事件
type BatchPricingCompletedEvent struct {
	BatchID        string    `json:"batch_id"`
	Symbols        []string  `json:"symbols"`
	TotalContracts int       `json:"total_contracts"`
	SuccessCount   int       `json:"success_count"`
	FailureCount   int       `json:"failure_count"`
	AverageTime    float64   `json:"average_time"`
	CompletedAt    int64     `json:"completed_at"`
	OccurredOn     time.Time `json:"occurred_on"`
}
