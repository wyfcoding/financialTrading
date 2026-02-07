package application

import "github.com/shopspring/decimal"

// ExecutionDTO 执行记录 DTO
type ExecutionDTO struct {
	ExecutionID string `json:"execution_id"`
	OrderID     string `json:"order_id"`
	Symbol      string `json:"symbol,omitempty"`
	Status      string `json:"status"`
	ExecutedQty string `json:"executed_qty"`
	ExecutedPx  string `json:"executed_px"`
	Timestamp   int64  `json:"timestamp"`
}

// ExecuteOrderCommand 执行订单命令
type ExecuteOrderCommand struct {
	OrderID  string          `json:"order_id" binding:"required"`
	UserID   string          `json:"user_id" binding:"required"`
	Symbol   string          `json:"symbol" binding:"required"`
	Side     string          `json:"side" binding:"required"`
	Price    decimal.Decimal `json:"price" binding:"required"`
	Quantity decimal.Decimal `json:"quantity" binding:"required"`
}

// SubmitAlgoCommand 提交算法订单命令
type SubmitAlgoCommand struct {
	UserID    string          `json:"user_id" binding:"required"`
	Symbol    string          `json:"symbol" binding:"required"`
	Side      string          `json:"side" binding:"required"`
	TotalQty  decimal.Decimal `json:"total_qty" binding:"required"`
	AlgoType  string          `json:"algo_type" binding:"required"` // TWAP, VWAP, POV, SOR
	StartTime int64           `json:"start_time"`
	EndTime   int64           `json:"end_time"`
	Params    string          `json:"params"`
}

// SubmitFIXOrderCommand FIX 订单命令
type SubmitFIXOrderCommand struct {
	ClOrdID  string          `json:"cl_ord_id" binding:"required"`
	UserID   string          `json:"user_id" binding:"required"`
	Symbol   string          `json:"symbol" binding:"required"`
	Side     string          `json:"side" binding:"required"`
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
}
