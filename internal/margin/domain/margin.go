//go:build margin_domain_v2
// +build margin_domain_v2

// 变更说明：重构 Margin 领域模型，支持多级阶梯保证金与动态追缴流程。
package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type MarginTier struct {
	MaxLeverage       decimal.Decimal `json:"max_leverage"`
	InitialMargin     decimal.Decimal `json:"initial_margin_rate"`
	MaintenanceMargin decimal.Decimal `json:"maintenance_margin_rate"`
	MaxPositionValue  decimal.Decimal `json:"max_position_value"`
}

type MarginAccount struct {
	AccountID         string          `json:"account_id"`
	Equity            decimal.Decimal `json:"equity"`             // 净值
	MarginBalance     decimal.Decimal `json:"margin_balance"`     // 保证金余额
	UsedMargin        decimal.Decimal `json:"used_margin"`        // 已用保证金
	MaintenanceMargin decimal.Decimal `json:"maintenance_margin"` // 维持保证金
	MarginLevel       decimal.Decimal `json:"margin_level"`       // 保证金水平 (Equity / UsedMargin)
	Leverage          decimal.Decimal `json:"leverage"`
	IsLiquidating     bool            `json:"is_liquidating"`
}

type MarginCall struct {
	ID             string          `json:"id"`
	AccountID      string          `json:"account_id"`
	RequiredAmount decimal.Decimal `json:"required_amount"`
	Deadline       time.Time       `json:"deadline"`
	Status         string          `json:"status"` // PENDING, PAID, LAPSED
}

var (
	ErrMarginLevelTooLow = errors.New("margin level below liquidation threshold")
	ErrMarginCallActive  = errors.New("existing active margin call")
)
