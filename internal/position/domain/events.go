package domain

import "time"

const (
	PositionCreatedEventType          = "PositionCreated"
	PositionUpdatedEventType          = "PositionUpdated"
	PositionClosedEventType           = "PositionClosed"
	PositionPnLUpdatedEventType       = "PositionPnLUpdated"
	PositionCostMethodChangedEventType = "PositionCostMethodChanged"
	PositionFlipEventType             = "PositionFlip"
)

// PositionCreatedEvent 头寸创建事件
type PositionCreatedEvent struct {
	UserID            string          `json:"user_id"`
	Symbol            string          `json:"symbol"`
	Quantity          float64         `json:"quantity"`
	AverageEntryPrice float64         `json:"average_entry_price"`
	Method            CostBasisMethod `json:"method"`
	OccurredOn        time.Time       `json:"occurred_on"`
}

// PositionUpdatedEvent 头寸更新事件
type PositionUpdatedEvent struct {
	UserID          string    `json:"user_id"`
	Symbol          string    `json:"symbol"`
	OldQuantity     float64   `json:"old_quantity"`
	NewQuantity     float64   `json:"new_quantity"`
	OldAveragePrice float64   `json:"old_average_price"`
	NewAveragePrice float64   `json:"new_average_price"`
	TradeSide       string    `json:"trade_side"`
	TradeQuantity   float64   `json:"trade_quantity"`
	TradePrice      float64   `json:"trade_price"`
	OccurredOn      time.Time `json:"occurred_on"`
}

// PositionClosedEvent 头寸关闭事件
type PositionClosedEvent struct {
	UserID        string    `json:"user_id"`
	Symbol        string    `json:"symbol"`
	FinalQuantity float64   `json:"final_quantity"`
	RealizedPnL   float64   `json:"realized_pnl"`
	ClosedAt      int64     `json:"closed_at"`
	OccurredOn    time.Time `json:"occurred_on"`
}

// PositionPnLUpdatedEvent 头寸盈亏更新事件
type PositionPnLUpdatedEvent struct {
	UserID         string    `json:"user_id"`
	Symbol         string    `json:"symbol"`
	OldRealizedPnL float64   `json:"old_realized_pnl"`
	NewRealizedPnL float64   `json:"new_realized_pnl"`
	TradeQuantity  float64   `json:"trade_quantity"`
	TradePrice     float64   `json:"trade_price"`
	PnLChange      float64   `json:"pnl_change"`
	UpdatedAt      int64     `json:"updated_at"`
	OccurredOn     time.Time `json:"occurred_on"`
}

// PositionCostMethodChangedEvent 头寸成本计算方法变更事件
type PositionCostMethodChangedEvent struct {
	UserID     string          `json:"user_id"`
	Symbol     string          `json:"symbol"`
	OldMethod  CostBasisMethod `json:"old_method"`
	NewMethod  CostBasisMethod `json:"new_method"`
	ChangedAt  int64           `json:"changed_at"`
	OccurredOn time.Time       `json:"occurred_on"`
}

// PositionFlipEvent 头寸反手事件
type PositionFlipEvent struct {
	UserID       string    `json:"user_id"`
	Symbol       string    `json:"symbol"`
	OldDirection string    `json:"old_direction"`
	NewDirection string    `json:"new_direction"`
	FlipQuantity float64   `json:"flip_quantity"`
	FlipPrice    float64   `json:"flip_price"`
	OccurredOn   time.Time `json:"occurred_on"`
}
