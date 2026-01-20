package application

import "github.com/shopspring/decimal"

// ExecuteOrderCommand 执行订单命令
type ExecuteOrderCommand struct {
	OrderID  string
	UserID   string
	Symbol   string
	Side     string
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

// SubmitAlgoCommand 提交算法订单命令
type SubmitAlgoCommand struct {
	UserID    string
	Symbol    string
	Side      string
	TotalQty  decimal.Decimal
	AlgoType  string
	StartTime int64
	EndTime   int64
	Params    string
}

// ExecutionDTO 执行结果数据传输对象
type ExecutionDTO struct {
	ExecutionID string
	OrderID     string
	Symbol      string
	Status      string
	ExecutedQty string
	ExecutedPx  string
	Timestamp   int64
}

// SubmitFIXOrderCommand来自 FIX 网关的订单请求
type SubmitFIXOrderCommand struct {
	UserID   string
	ClOrdID  string
	Symbol   string
	Side     string
	Price    decimal.Decimal
	Quantity decimal.Decimal
}
