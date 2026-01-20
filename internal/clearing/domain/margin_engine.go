package domain

import (
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algorithm/finance"
)

// Position 简化版持仓快照
type Position struct {
	Symbol   string
	Side     string
	Quantity float64
	Price    float64
}

// RiskParameter 各资产的风控参数
type RiskParameter struct {
	BaseMarginRate        float64
	MaintenanceMarginRate float64
}

// MarginRequirement 最终保证金计算结果
type MarginRequirement struct {
	InitialMargin     decimal.Decimal
	MaintenanceMargin decimal.Decimal
	RiskScore         float64
}

// PortfolioMarginEngine 组合保证金引擎
type PortfolioMarginEngine struct {
	impl *finance.PortfolioMarginEngine
}

func NewPortfolioMarginEngine(correlations map[string]map[string]float64) *PortfolioMarginEngine {
	return &PortfolioMarginEngine{
		impl: &finance.PortfolioMarginEngine{CorrelationMatrix: correlations},
	}
}

// CalculateMargin 计算整个组合的保证金要求
func (e *PortfolioMarginEngine) CalculateMargin(positions []Position, params map[string]RiskParameter) (*MarginRequirement, error) {
	pkgPositions := make([]finance.PositionData, len(positions))
	for i, p := range positions {
		pkgPositions[i] = finance.PositionData{
			Symbol:   p.Symbol,
			Side:     p.Side,
			Quantity: p.Quantity,
			Price:    p.Price,
		}
	}

	rates := make(map[string]float64)
	for k, v := range params {
		rates[k] = v.BaseMarginRate
	}

	im, mm, score := e.impl.Calculate(pkgPositions, rates)

	return &MarginRequirement{
		InitialMargin:     im,
		MaintenanceMargin: mm,
		RiskScore:         score,
	}, nil
}

// AccountHealth 账户风险健康度
type AccountHealth struct {
	UserID            string
	TotalEquity       decimal.Decimal
	MaintenanceMargin decimal.Decimal
	HealthScale       float64 // 1.0 = OK, < 1.0 = Risk of liquidation
}

func (h *AccountHealth) NeedsLiquidation() bool {
	return h.HealthScale < 1.0
}

// LiquidationTrigger 强平触发指令
type LiquidationTrigger struct {
	UserID    string
	Symbol    string
	Side      string
	Quantity  float64
	Reason    string
	Timestamp int64
}

// CheckAccountHealth 核查账户健康度逻辑
func (e *PortfolioMarginEngine) CheckAccountHealth(userID string, totalEquity decimal.Decimal, positions []Position, params map[string]RiskParameter) (*AccountHealth, error) {
	req, err := e.CalculateMargin(positions, params)
	if err != nil {
		return nil, err
	}

	scale := 1.0
	if !req.MaintenanceMargin.IsZero() {
		scale = totalEquity.InexactFloat64() / req.MaintenanceMargin.InexactFloat64()
	}

	return &AccountHealth{
		UserID:            userID,
		TotalEquity:       totalEquity,
		MaintenanceMargin: req.MaintenanceMargin,
		HealthScale:       scale,
	}, nil
}
