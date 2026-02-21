//go:build matching_experimental
// +build matching_experimental

package domain

import (
	"github.com/shopspring/decimal"
)

type MatchTaskType int

const (
	TaskNewOrder MatchTaskType = iota
	TaskCancelOrder
	TaskExecuteAuction
)

// MatchTask 是 RingBuffer 定序后的最小处理单元
type MatchTask struct {
	Type     MatchTaskType
	Order    *Order
	OrderID  uint64
	Symbol   string
	Side     Side
	Price    decimal.Decimal
	Quantity decimal.Decimal

	// 用于同步返回结果
	ResultChan chan any
}
