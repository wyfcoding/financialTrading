package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	PositionCreatedEventType           = "PositionCreated"
	PositionUpdatedEventType           = "PositionUpdated"
	PositionClosedEventType            = "PositionClosed"
	PositionPnLUpdatedEventType        = "PositionPnLUpdated"
	PositionCostMethodChangedEventType = "PositionCostMethodChanged"
	PositionFlipEventType              = "PositionFlip"
	PositionTccTryFrozenEventType      = "position.tcc.try_freeze"
	PositionTccConfirmedEventType      = "position.tcc.confirm_freeze"
	PositionTccCanceledEventType       = "position.tcc.cancel_freeze"
	PositionSagaDeductedEventType      = "position.saga.deduct_frozen"
	PositionSagaRefundedEventType      = "position.saga.refund_frozen"
)

// PositionCreatedEvent 头寸创建事件
type PositionCreatedEvent struct {
	UserID            string          `json:"user_id"`
	Symbol            string          `json:"symbol"`
	Quantity          decimal.Decimal `json:"quantity"`
	AverageEntryPrice decimal.Decimal `json:"average_entry_price"`
	Method            CostBasisMethod `json:"method"`
	OccurredOn        time.Time       `json:"occurred_on"`
}

// PositionUpdatedEvent 头寸更新事件
type PositionUpdatedEvent struct {
	UserID          string          `json:"user_id"`
	Symbol          string          `json:"symbol"`
	OldQuantity     decimal.Decimal `json:"old_quantity"`
	NewQuantity     decimal.Decimal `json:"new_quantity"`
	OldAveragePrice decimal.Decimal `json:"old_average_price"`
	NewAveragePrice decimal.Decimal `json:"new_average_price"`
	TradeSide       string          `json:"trade_side"`
	TradeQuantity   decimal.Decimal `json:"trade_quantity"`
	TradePrice      decimal.Decimal `json:"trade_price"`
	OccurredOn      time.Time       `json:"occurred_on"`
}

// PositionClosedEvent 头寸关闭事件
type PositionClosedEvent struct {
	UserID        string          `json:"user_id"`
	Symbol        string          `json:"symbol"`
	FinalQuantity decimal.Decimal `json:"final_quantity"`
	RealizedPnL   decimal.Decimal `json:"realized_pnl"`
	ClosedAt      int64           `json:"closed_at"`
	OccurredOn    time.Time       `json:"occurred_on"`
}

// PositionPnLUpdatedEvent 头寸盈亏更新事件
type PositionPnLUpdatedEvent struct {
	UserID         string          `json:"user_id"`
	Symbol         string          `json:"symbol"`
	OldRealizedPnL decimal.Decimal `json:"old_realized_pnl"`
	NewRealizedPnL decimal.Decimal `json:"new_realized_pnl"`
	TradeQuantity  decimal.Decimal `json:"trade_quantity"`
	TradePrice     decimal.Decimal `json:"trade_price"`
	PnLChange      decimal.Decimal `json:"pnl_change"`
	UpdatedAt      int64           `json:"updated_at"`
	OccurredOn     time.Time       `json:"occurred_on"`
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
	UserID       string          `json:"user_id"`
	Symbol       string          `json:"symbol"`
	OldDirection string          `json:"old_direction"`
	NewDirection string          `json:"new_direction"`
	FlipQuantity decimal.Decimal `json:"flip_quantity"`
	FlipPrice    decimal.Decimal `json:"flip_price"`
	OccurredOn   time.Time       `json:"occurred_on"`
}
