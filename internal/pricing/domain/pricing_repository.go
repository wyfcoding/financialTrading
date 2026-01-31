package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// MarketDataClient 市场数据客户端接口
type MarketDataClient interface {
	GetPrice(ctx context.Context, symbol string) (decimal.Decimal, error)
}

// PricingRepository 定价服务统一仓储接口
type PricingRepository interface {
	// Price (Simple Asset Price)
	SavePrice(ctx context.Context, price *Price) error
	GetLatestPrice(ctx context.Context, symbol string) (*Price, error)
	ListLatestPrices(ctx context.Context, symbols []string) ([]*Price, error)

	// PricingResult (Option/Derivatives Pricing)
	SavePricingResult(ctx context.Context, result *PricingResult) error
	GetLatestPricingResult(ctx context.Context, symbol string) (*PricingResult, error)
	GetPricingResultHistory(ctx context.Context, symbol string, limit int) ([]*PricingResult, error)
}
