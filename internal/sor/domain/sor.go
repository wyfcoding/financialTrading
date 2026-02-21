//go:build sor_legacy_min
// +build sor_legacy_min

// 变更说明：增强 SOR 领域模型。
// 智能路由不仅要考虑价格，还要考虑流动性深度、延迟、佣金成本及交易所返佣（Rebates）。
package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

type ExecutionVenue struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Latency     int64           `json:"latency_ns"` // 历史平均延迟
	FeeRate     decimal.Decimal `json:"fee_rate"`
	Reliability decimal.Decimal `json:"reliability"`
}

type RouteSegment struct {
	VenueID           string          `json:"venue_id"`
	Quantity          decimal.Decimal `json:"quantity"`
	Price             decimal.Decimal `json:"price"`
	EstimatedSlippage decimal.Decimal `json:"estimated_slippage"`
}

type RoutingStrategy interface {
	// FindBestRoutes 根据当前全市场深度（NBBO）计算最优分割方案。
	FindBestRoutes(ctx context.Context, symbol string, qty decimal.Decimal, side string) ([]RouteSegment, error)
}

type OrderChunk struct {
	ID       string
	ParentID string
	VenueID  string
	Quantity decimal.Decimal
	Price    decimal.Decimal
	Status   string
}
