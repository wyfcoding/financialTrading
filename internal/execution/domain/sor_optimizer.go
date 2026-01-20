package domain

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algorithm/optimization"
)

// SOROptimizer 封装了智能路由的优化计算逻辑
type SOROptimizer struct {
	impl *optimization.SOROptimizer
}

func NewSOROptimizer(latencyFactor float64) *SOROptimizer {
	return &SOROptimizer{
		impl: &optimization.SOROptimizer{LatencyFactor: latencyFactor},
	}
}

// RouteResult 路由建议结果
type RouteResult struct {
	VenueID  string
	Quantity decimal.Decimal
	Price    decimal.Decimal
}

// Optimize 计算最优的订单分配方案
func (o *SOROptimizer) Optimize(
	ctx context.Context,
	side TradeSide,
	totalQty decimal.Decimal,
	venues []*Venue,
	depths []*VenueDepth,
) ([]RouteResult, error) {
	var inputs []optimization.RouteInput
	venueMap := make(map[string]*Venue)
	for _, v := range venues {
		venueMap[v.ID] = v
	}

	for _, d := range depths {
		v := venueMap[d.VenueID]
		levels := d.Asks
		if side == TradeSideSell {
			levels = d.Bids
		}
		for _, l := range levels {
			inputs = append(inputs, optimization.RouteInput{
				VenueID:   d.VenueID,
				Price:     l.Price,
				Quantity:  l.Quantity,
				FeeRate:   v.ExecutionFee,
				LatencyMs: float64(v.Latency.Milliseconds()),
			})
		}
	}

	outputs := o.impl.Optimize(totalQty, inputs, side == TradeSideBuy)

	var results []RouteResult
	for _, out := range outputs {
		results = append(results, RouteResult{
			VenueID:  out.VenueID,
			Quantity: out.Quantity,
			Price:    out.Price,
		})
	}

	return results, nil
}

// End of SOR Optimizer
