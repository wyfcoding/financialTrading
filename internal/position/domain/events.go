package domain

import (
	"time"
)

// PositionCreatedEvent 头寸创建事件
type PositionCreatedEvent struct {
	UserID            string
	Symbol            string
	Quantity          float64
	AverageEntryPrice float64
	Method            CostBasisMethod
	OccurredOn        time.Time
}

// PositionUpdatedEvent 头寸更新事件
type PositionUpdatedEvent struct {
	UserID            string
	Symbol            string
	OldQuantity       float64
	NewQuantity       float64
	OldAveragePrice   float64
	NewAveragePrice   float64
	TradeSide         string
	TradeQuantity     float64
	TradePrice        float64
	OccurredOn        time.Time
}

// PositionClosedEvent 头寸关闭事件
type PositionClosedEvent struct {
	UserID            string
	Symbol            string
	FinalQuantity     float64
	RealizedPnL       float64
	ClosedAt          int64
	OccurredOn        time.Time
}

// PositionPnLUpdatedEvent 头寸盈亏更新事件
type PositionPnLUpdatedEvent struct {
	UserID            string
	Symbol            string
	OldRealizedPnL    float64
	NewRealizedPnL    float64
	TradeQuantity     float64
	TradePrice        float64
	PnLChange         float64
	UpdatedAt         int64
	OccurredOn        time.Time
}

// PositionCostMethodChangedEvent 头寸成本计算方法变更事件
type PositionCostMethodChangedEvent struct {
	UserID            string
	Symbol            string
	OldMethod         CostBasisMethod
	NewMethod         CostBasisMethod
	ChangedAt         int64
	OccurredOn        time.Time
}

// PositionFlipEvent 头寸反手事件
type PositionFlipEvent struct {
	UserID            string
	Symbol            string
	OldDirection      string
	NewDirection      string
	FlipQuantity      float64
	FlipPrice         float64
	OccurredOn        time.Time
}
