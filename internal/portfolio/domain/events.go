// 变更说明：新增 portfolio 领域事件定义。
// 投资组合服务负责资产分配、绩效归因和再平衡，所有状态变更均需通过事件发布。
package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	EventPortfolioCreated    = "portfolio.created"
	EventPositionOpened      = "portfolio.position.opened"
	EventPositionClosed      = "portfolio.position.closed"
	EventPositionAdjusted    = "portfolio.position.adjusted"
	EventPortfolioRebalanced = "portfolio.rebalanced"
	EventRiskMetricsUpdated  = "portfolio.risk.updated"
	EventPerformanceCalculated = "portfolio.performance.calculated"
	EventBenchmarkAssigned   = "portfolio.benchmark.assigned"
)

type PortfolioEvent struct {
	EventID       string            `json:"event_id"`
	EventType     string            `json:"event_type"`
	PortfolioID   string            `json:"portfolio_id"`
	UserID        string            `json:"user_id"`
	TransactionID string            `json:"transaction_id"`
	OccurredAt    time.Time         `json:"occurred_at"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type PositionAdjustedEvent struct {
	PortfolioEvent
	Symbol        string          `json:"symbol"`
	QuantityDelta decimal.Decimal `json:"quantity_delta"`
	Price         decimal.Decimal `json:"price"`
	NewQuantity   decimal.Decimal `json:"new_quantity"`
	NewAvgCost    decimal.Decimal `json:"new_avg_cost"`
	RealizedPnL   decimal.Decimal `json:"realized_pnl"`
}

type RiskMetricsUpdatedEvent struct {
	PortfolioEvent
	Volatility  decimal.Decimal `json:"volatility"`
	SharpeRatio decimal.Decimal `json:"sharpe_ratio"`
	MaxDrawdown decimal.Decimal `json:"max_drawdown"`
	VaR95       decimal.Decimal `json:"var_95"`
}
