package application

import (
	"time"

	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
)

// PriceOptionCommand 期权定价命令
type PriceOptionCommand struct {
	Symbol          string
	OptionType      string
	StrikePrice     float64
	ExpiryDate      int64
	UnderlyingPrice float64
	Volatility      float64
	RiskFreeRate    float64
	DividendYield   float64
	PricingModel    string
}

// UpdateVolatilityCommand 更新波动率命令
type UpdateVolatilityCommand struct {
	Symbol        string
	NewVolatility float64
	Reason        string
}

// ChangePricingModelCommand 变更定价模型命令
type ChangePricingModelCommand struct {
	Symbol   string
	NewModel string
}

// BatchPriceOptionsCommand 批量定价命令
type BatchPriceOptionsCommand struct {
	Contracts []PriceOptionCommand
	BatchID   string
}

// BatchPricingResult 批量定价结果
type BatchPricingResult struct {
	BatchID      string
	Results      []*domain.PricingResult
	SuccessCount int
	FailureCount int
	AverageTime  float64
}

// PriceDTO 价格 DTO
type PriceDTO struct {
	Symbol    string    `json:"symbol"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	Mid       float64   `json:"mid"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
}
