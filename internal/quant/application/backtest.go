package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/analytics"
)

type BacktestRequest struct {
	Symbol         string
	StartTime      time.Time
	EndTime        time.Time
	InitialBalance decimal.Decimal
}

type BacktestResult struct {
	TotalPnL     decimal.Decimal
	MaxDrawdown  decimal.Decimal
	TradesCount  int
	FinalBalance decimal.Decimal
}

type BacktestEngine struct {
	fetcher *analytics.ClickHouseFetcher
}

func NewBacktestEngine(fetcher *analytics.ClickHouseFetcher) *BacktestEngine {
	return &BacktestEngine{fetcher: fetcher}
}

func (e *BacktestEngine) Run(ctx context.Context, req BacktestRequest) (*BacktestResult, error) {
	// 1. 获取历史数据
	quotes, err := e.fetcher.FetchQuotes(ctx, req.Symbol, req.StartTime, req.EndTime, 10000)
	if err != nil {
		return nil, err
	}

	balance := req.InitialBalance
	pnl := decimal.Zero
	trades := 0

	// 模拟简单的趋势跟踪策略
	// 此处仅为演示引擎流程，实际应由策略接口实现
	var lastPrice decimal.Decimal
	for i, q := range quotes {
		if i == 0 {
			lastPrice = q.LastPrice
			continue
		}

		// 简单逻辑：价格上涨买入，下跌卖出
		if q.LastPrice.GreaterThan(lastPrice) {
			// Buy logic simulator
			trades++
		} else if q.LastPrice.LessThan(lastPrice) {
			// Sell logic simulator
			trades++
		}

		lastPrice = q.LastPrice
	}

	return &BacktestResult{
		TotalPnL:     pnl,
		MaxDrawdown:  decimal.Zero,
		TradesCount:  trades,
		FinalBalance: balance.Add(pnl),
	}, nil
}
