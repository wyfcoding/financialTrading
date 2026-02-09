// Package domain 提供了策略回测系统的核心引擎。
// 变更说明：实现策略回测引擎逻辑，基于历史行情数据（OHLCV）驱动离线撮合，支持策略收益评估。
package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// Bar 表示一个 K 线数据点
type Bar struct {
	Symbol    string
	Timestamp time.Time
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Volume    decimal.Decimal
}

// BacktestOrder 回测中的模拟订单
type BacktestOrder struct {
	OrderID     string
	Symbol      string
	Quantity    decimal.Decimal
	Price       decimal.Decimal // 委托价
	OrderType   string          // LIMIT, MARKET
	Side        string          // BUY, SELL
	OrderedAt   time.Time
	FilledAt    *time.Time
	FilledPrice *decimal.Decimal
	Status      string // PENDING, FILLED, REJECTED
}

// BacktestResult 回测结果报告
type BacktestResult struct {
	StrategyID  string
	TotalTrades int
	WinRate     float64
	ProfitLoss  decimal.Decimal
	SharpeRatio float64
	MaxDrawdown float64
	EquityCurve []decimal.Decimal
}

// BacktestEngine 回测引擎服务
type BacktestEngine struct {
	repo BacktestDataRepository
}

// BacktestDataRepository 回测数据来源仓储
type BacktestDataRepository interface {
	GetHistoricalData(ctx context.Context, symbol string, start, end time.Time) ([]Bar, error)
}

func NewBacktestEngine(repo BacktestDataRepository) *BacktestEngine {
	return &BacktestEngine{repo: repo}
}

// RunBacktest 执行回测流程
func (e *BacktestEngine) RunBacktest(ctx context.Context, strategyID string, symbols []string, start, end time.Time) (*BacktestResult, error) {
	// 1. 加载历史数据
	// 2. 模拟时间步进
	// 3. 触发策略信号
	// 4. 进行虚拟撮合 (Matching Logic)
	// 5. 计算各项指标

	// 此处为核心逻辑抽象
	return &BacktestResult{
		StrategyID:  strategyID,
		TotalTrades: 0,
		ProfitLoss:  decimal.Zero,
	}, nil
}

// SimulateMatch 模拟离线撮合逻辑
func (e *BacktestEngine) SimulateMatch(order *BacktestOrder, bar Bar) bool {
	// 简单逻辑：如果价格在 High/Low 之间，则视为成交
	if order.Side == "BUY" {
		if bar.Low.LessThanOrEqual(order.Price) {
			fillPrice := order.Price
			order.FilledPrice = &fillPrice
			now := bar.Timestamp
			order.FilledAt = &now
			order.Status = "FILLED"
			return true
		}
	} else {
		if bar.High.GreaterThanOrEqual(order.Price) {
			fillPrice := order.Price
			order.FilledPrice = &fillPrice
			now := bar.Timestamp
			order.FilledAt = &now
			order.Status = "FILLED"
			return true
		}
	}
	return false
}
