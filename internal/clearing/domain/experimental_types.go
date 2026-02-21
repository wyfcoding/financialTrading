//go:build clearing_experimental
// +build clearing_experimental

package domain

import (
	"context"
	"time"
)

// Position 实验分支统一持仓模型，供保证金与违约流程共享。
type Position struct {
	ID          string  `json:"id"`
	MemberID    string  `json:"member_id"`
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"`
	Quantity    float64 `json:"quantity"`
	Price       float64 `json:"price"`
	MarketValue float64 `json:"market_value"`
}

// Portfolio 实验分支统一组合模型。
type Portfolio struct {
	ID        string      `json:"id"`
	MemberID  string      `json:"member_id"`
	Positions []*Position `json:"positions"`
}

// RiskModel 风险权重模型。
type RiskModel interface {
	CalculateRiskWeight(symbol string, marketValue float64) float64
}

// SettlementStatus 违约流程需要的结算状态。
type SettlementStatus string

const (
	SettlementPending SettlementStatus = "PENDING"
	SettlementSettled SettlementStatus = "SETTLED"
	SettlementFailed  SettlementStatus = "FAILED"
)

// SettlementInstruction 违约流程需要的最小结算指令模型。
type SettlementInstruction struct {
	ID     string           `json:"id"`
	Status SettlementStatus `json:"status"`
}

// SettlementRepository 违约处理依赖的结算查询接口。
type SettlementRepository interface {
	GetInstruction(ctx context.Context, id string) (*SettlementInstruction, error)
}

func timePtrNow() *time.Time {
	now := time.Now()
	return &now
}
