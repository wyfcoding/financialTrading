package domain

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// StressScenario 压力测试场景
type StressScenario struct {
	Name        string
	Description string
	PriceShift  map[string]float64 // Symbol -> Percentage Shift (e.g., -0.20 for 20% drop)
}

// StressTestEngine 压力测试引擎
type StressTestEngine struct {
	scenarios map[string]*StressScenario
}

func NewStressTestEngine() *StressTestEngine {
	e := &StressTestEngine{
		scenarios: make(map[string]*StressScenario),
	}
	e.initDefaultScenarios()
	return e
}

func (e *StressTestEngine) initDefaultScenarios() {
	// 全球金融危机场景 (GFC)
	e.scenarios["GFC"] = &StressScenario{
		Name:        "Global Financial Crisis",
		Description: "Market wide crash, high volatility",
		PriceShift: map[string]float64{
			"DEFAULT": -0.40, // 默认下跌 40%
			"GOLD":    0.10,  // 避险资产上涨 10%
		},
	}

	// 闪崩场景 (Flash Crash)
	e.scenarios["FLASH_CRASH"] = &StressScenario{
		Name:        "Flash Crash",
		Description: "Sudden 10% drop in index within minutes",
		PriceShift: map[string]float64{
			"DEFAULT": -0.15,
		},
	}
}

// RunScenario 在指定资产组合上运行压力测试
func (e *StressTestEngine) RunScenario(scenarioName string, assets []PortfolioAsset) (*StressTestResult, error) {
	scenario, ok := e.scenarios[scenarioName]
	if !ok {
		return nil, fmt.Errorf("scenario %s not found", scenarioName)
	}

	var totalInitialValue decimal.Decimal
	var totalStressedValue decimal.Decimal

	for _, asset := range assets {
		initialVal := asset.Position.Mul(asset.CurrentPrice)
		totalInitialValue = totalInitialValue.Add(initialVal)

		shift, found := scenario.PriceShift[asset.Symbol]
		if !found {
			shift = scenario.PriceShift["DEFAULT"]
		}

		stressedPrice := asset.CurrentPrice.Mul(decimal.NewFromFloat(1.0 + shift))
		stressedVal := asset.Position.Mul(stressedPrice)
		totalStressedValue = totalStressedValue.Add(stressedVal)
	}

	pnl := totalStressedValue.Sub(totalInitialValue)
	percentage := decimal.Zero
	if !totalInitialValue.IsZero() {
		percentage = pnl.Div(totalInitialValue).Mul(decimal.NewFromInt(100))
	}

	return &StressTestResult{
		ScenarioName:   scenario.Name,
		PnLImpact:      pnl,
		PercentageDrop: percentage,
		Survived:       percentage.Abs().LessThan(decimal.NewFromInt(50)), // 示例：亏损超过 50% 视为未通过
	}, nil
}

// StressTestResult 压力测试报告
type StressTestResult struct {
	ScenarioName   string
	PnLImpact      decimal.Decimal
	PercentageDrop decimal.Decimal
	Survived       bool
}
