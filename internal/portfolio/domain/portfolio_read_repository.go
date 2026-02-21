// 变更说明：新增 portfolio 读模型接口，支持多维度的绩效分析和风险看板。
package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type PortfolioReadRepository interface {
	// GetPortfolioView 获取组合概览快照。
	GetPortfolioView(ctx context.Context, portfolioID string) (*PortfolioView, error)
	// GetUserPerformance 获取用户整体绩效指标。
	GetUserPerformance(ctx context.Context, userID string) (*PerformanceView, error)
	// GetHistoricalReturns 获取历史收益率曲线。
	GetHistoricalReturns(ctx context.Context, portfolioID string, start, end time.Time) ([]*ReturnPoint, error)
}

type PortfolioView struct {
	PortfolioID    string          `json:"portfolio_id"`
	TotalValue     decimal.Decimal `json:"total_value"`
	UnrealizedPnL  decimal.Decimal `json:"unrealized_pnl"`
	RealizedPnL    decimal.Decimal `json:"realized_pnl"`
	DayReturn      decimal.Decimal `json:"day_return"`
	DayReturnPct   decimal.Decimal `json:"day_return_pct"`
	AssetAllocation map[string]decimal.Decimal `json:"asset_allocation"`
	Positions      []*PositionView `json:"positions"`
}

type PositionView struct {
	Symbol        string          `json:"symbol"`
	Quantity      decimal.Decimal `json:"quantity"`
	MarketValue   decimal.Decimal `json:"market_value"`
	Weight        decimal.Decimal `json:"weight"`
	PnL           decimal.Decimal `json:"pnl"`
	PnLPct        decimal.Decimal `json:"pnl_pct"`
}

type PerformanceView struct {
	TotalReturn    decimal.Decimal `json:"total_return"`
	AnnualizedReturn decimal.Decimal `json:"annualized_return"`
	SharpeRatio    decimal.Decimal `json:"sharpe_ratio"`
	MaxDrawdown    decimal.Decimal `json:"max_drawdown"`
	WinRate        decimal.Decimal `json:"win_rate"`
}

type ReturnPoint struct {
	Timestamp time.Time       `json:"timestamp"`
	Value     decimal.Decimal `json:"value"`
	Return    decimal.Decimal `json:"return"`
}
