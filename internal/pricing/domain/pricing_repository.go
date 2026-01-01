package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// MarketDataClient 市场数据客户端接口
type MarketDataClient interface {
	GetPrice(ctx context.Context, symbol string) (decimal.Decimal, error)
}

// PricingRepository 定价历史仓储接口
type PricingRepository interface {
	Save(ctx context.Context, result *PricingResult) error
	GetLatest(ctx context.Context, symbol string) (*PricingResult, error)
	GetHistory(ctx context.Context, symbol string, limit int) ([]*PricingResult, error)
}
