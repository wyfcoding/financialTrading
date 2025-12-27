// Package domain 撮合引擎的领域模型
package domain

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algorithm"
	"gorm.io/gorm"
)

// MatchingResult 撮合结果
// 包含撮合后的成交信息和订单状态
type MatchingResult struct {
	OrderID           string             // 订单 ID
	Trades            []*algorithm.Trade // 成交列表
	RemainingQuantity decimal.Decimal    // 剩余数量
	Status            string             // 状态 (MATCHED, PARTIALLY_MATCHED)
}

// OrderBookSnapshot 订单簿快照
type OrderBookSnapshot struct {
	gorm.Model
	// Symbol 交易对
	Symbol string `gorm:"column:symbol;type:varchar(20);index;not null"`
	// Bids 买单
	Bids []*OrderBookLevel `gorm:"-"`
	// Asks 卖单
	Asks []*OrderBookLevel `gorm:"-"`
	// BidsJSON 序列化后的买单
	BidsJSON string `gorm:"column:bids;type:text"`
	// AsksJSON 序列化后的卖单
	AsksJSON string `gorm:"column:asks;type:text"`
	// Timestamp 时间戳
	Timestamp int64 `gorm:"column:timestamp;type:bigint"`
}

// OrderBookLevel 订单簿层级
type OrderBookLevel struct {
	// Price 价格
	Price decimal.Decimal `json:"price"`
	// Quantity 数量
	Quantity decimal.Decimal `json:"quantity"`
}

// MatchingEngine 撮合引擎接口
type MatchingEngine interface {
	// SubmitOrder 提交订单进行撮合
	SubmitOrder(order *algorithm.Order) *MatchingResult
	// GetOrderBook 获取订单簿快照
	GetOrderBook(symbol string, depth int) *OrderBookSnapshot
	// GetTrades 获取成交历史
	GetTrades(symbol string, limit int) []*algorithm.Trade
}

// TradeModel 成交记录仓储模型
type TradeModel struct {
	gorm.Model
	TradeID      string          `gorm:"column:trade_id;type:varchar(32);uniqueIndex;not null"`
	OrderID      string          `gorm:"column:order_id;type:varchar(32);index;not null"`
	MatchOrderID string          `gorm:"column:match_order_id;type:varchar(32);index;not null"`
	Symbol       string          `gorm:"column:symbol;type:varchar(20);index;not null"`
	Side         string          `gorm:"column:side;type:varchar(10);not null"`
	Price        decimal.Decimal `gorm:"column:price;type:decimal(32,18);not null"`
	Quantity     decimal.Decimal `gorm:"column:quantity;type:decimal(32,18);not null"`
	ExecutedAt   int64           `gorm:"column:executed_at;type:bigint"`
}

// TradeRepository 成交记录仓储接口
type TradeRepository interface {
	// Save 保存成交记录
	Save(ctx context.Context, trade *algorithm.Trade) error
	// GetTradeHistory 获取成交历史
	GetTradeHistory(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error)
	// GetLatestTrades 获取最新成交
	GetLatestTrades(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error)
}

// OrderBookRepository 订单簿仓储接口
type OrderBookRepository interface {
	// SaveSnapshot 保存订单簿快照
	SaveSnapshot(ctx context.Context, snapshot *OrderBookSnapshot) error
	// GetLatestOrderBook 获取最新订单簿
	GetLatestOrderBook(ctx context.Context, symbol string) (*OrderBookSnapshot, error)
}
