// Package domain 包含撮合引擎的领域模型
package domain

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/pkg/algos"
)

// MatchingResult 撮合结果
// 包含撮合后的成交信息和订单状态
type MatchingResult struct {
	OrderID           string          // 订单 ID
	Trades            []*algos.Trade  // 成交列表
	RemainingQuantity decimal.Decimal // 剩余数量
	Status            string          // 状态 (MATCHED, PARTIALLY_MATCHED)
}

// OrderBookSnapshot 订单簿快照
type OrderBookSnapshot struct {
	// 交易对
	Symbol string
	// 买单
	Bids []*OrderBookLevel
	// 卖单
	Asks []*OrderBookLevel
	// 时间戳
	Timestamp int64
}

// OrderBookLevel 订单簿层级
type OrderBookLevel struct {
	// 价格
	Price decimal.Decimal
	// 数量
	Quantity decimal.Decimal
}

// MatchingEngine 撮合引擎接口
type MatchingEngine interface {
	// 提交订单进行撮合
	SubmitOrder(order *algos.Order) *MatchingResult
	// 获取订单簿快照
	GetOrderBook(symbol string, depth int) *OrderBookSnapshot
	// 获取成交历史
	GetTrades(symbol string, limit int) []*algos.Trade
}

// TradeRepository 成交记录仓储接口
type TradeRepository interface {
	// 保存成交记录
	Save(ctx context.Context, trade *algos.Trade) error
	// 获取成交历史
	GetHistory(ctx context.Context, symbol string, limit int) ([]*algos.Trade, error)
	// 获取最新成交
	GetLatest(ctx context.Context, symbol string, limit int) ([]*algos.Trade, error)
}

// OrderBookRepository 订单簿仓储接口
type OrderBookRepository interface {
	// 保存订单簿快照
	SaveSnapshot(ctx context.Context, snapshot *OrderBookSnapshot) error
	// 获取最新订单簿
	GetLatest(ctx context.Context, symbol string) (*OrderBookSnapshot, error)
}
