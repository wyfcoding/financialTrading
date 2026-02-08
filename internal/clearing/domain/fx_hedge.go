package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// FXHedgeEngine 汇率对冲引擎
type FXHedgeEngine struct {
	// 基础币种 (默认记账币种，如 USD)
	BaseCurrency string
}

func NewFXHedgeEngine(baseCurrency string) *FXHedgeEngine {
	return &FXHedgeEngine{
		BaseCurrency: baseCurrency,
	}
}

// FXExposure 币种敞口
type FXExposure struct {
	Currency    string          `json:"currency"`
	NetPrice    decimal.Decimal `json:"net_price"`
	NetQuantity decimal.Decimal `json:"net_quantity"`
	NetAmount   decimal.Decimal `json:"net_amount"` // 在该币种下的净盈亏/收支
}

// HedgeInstruction 对冲指令
type HedgeInstruction struct {
	Currency string          `json:"currency"`
	Side     string          `json:"side"` // BUY/SELL
	Amount   decimal.Decimal `json:"amount"`
}

// CalculateExposure 计算净敞口
func (e *FXHedgeEngine) CalculateExposure(nettingResults map[string]map[string]map[string]*NettingResult) map[string]*FXExposure {
	exposures := make(map[string]*FXExposure)

	for _, symbols := range nettingResults {
		for _, currencies := range symbols {
			for currency, res := range currencies {
				if currency == e.BaseCurrency {
					continue
				}

				if _, ok := exposures[currency]; !ok {
					exposures[currency] = &FXExposure{
						Currency: currency,
					}
				}
				exp := exposures[currency]
				exp.NetAmount = exp.NetAmount.Add(res.NetAmount)
			}
		}
	}

	return exposures
}

// GenerateHedgeInstructions 生成对冲指令
func (e *FXHedgeEngine) GenerateHedgeInstructions(exposures map[string]*FXExposure) []*HedgeInstruction {
	instructions := make([]*HedgeInstruction, 0)

	for currency, exp := range exposures {
		if exp.NetAmount.IsZero() {
			continue
		}

		// 如果 NetAmount 为负，说明该币种应付(Short)，需要买入(BUY)该币种进行对冲
		// 如果 NetAmount 为正，说明该币种应收(Long)，需要卖出(SELL)该币种兑换回 BaseCurrency
		side := "BUY"
		if exp.NetAmount.IsPositive() {
			side = "SELL"
		}

		instructions = append(instructions, &HedgeInstruction{
			Currency: currency,
			Side:     side,
			Amount:   exp.NetAmount.Abs(),
		})
	}

	return instructions
}

// FXHedgeExecutedEvent 汇率对冲执行事件
type FXHedgeExecutedEvent struct {
	Currency  string          `json:"currency"`
	Side      string          `json:"side"`
	Amount    decimal.Decimal `json:"amount"`
	Timestamp time.Time       `json:"timestamp"`
}
