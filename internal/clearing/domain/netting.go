//go:build !clearing_experimental
// +build !clearing_experimental

// 变更说明：实现多边净额结算(Multilateral Netting)算法。
// 核心逻辑：在一个清算周期内，将同一账户、同一币种、同一品种的所有买单和卖单进行轧差。
// 优点：极大减少 DVP（银货对付）次数，降低账户流水负担。
package domain

import (
	"github.com/shopspring/decimal"
)

type NettingEngine struct{}

type Trade struct {
	Symbol   string
	Side     string
	Quantity decimal.Decimal
	Price    decimal.Decimal
}

type PositionImpact struct {
	Symbol   string
	Quantity decimal.Decimal // 净数量变动（可正可负）
	CashFlow decimal.Decimal // 净资金变动（支出为负，收入为正）
}

func NewNettingEngine() *NettingEngine {
	return &NettingEngine{}
}

func (e *NettingEngine) CalculateNetting(trades []Trade) map[string]*PositionImpact {
	results := make(map[string]*PositionImpact)

	for _, t := range trades {
		if _, ok := results[t.Symbol]; !ok {
			results[t.Symbol] = &PositionImpact{Symbol: t.Symbol}
		}

		impact := results[t.Symbol]
		if t.Side == "BUY" {
			impact.Quantity = impact.Quantity.Add(t.Quantity)
			impact.CashFlow = impact.CashFlow.Sub(t.Price.Mul(t.Quantity))
		} else {
			impact.Quantity = impact.Quantity.Sub(t.Quantity)
			impact.CashFlow = impact.CashFlow.Add(t.Price.Mul(t.Quantity))
		}
	}
	return results
}

// CalculateMultilateralNetting 聚合用户/品种/币种维度的净额结果，供 FX 对冲流程使用。
func (e *NettingEngine) CalculateMultilateralNetting(settlements []*Settlement) map[string]map[string]map[string]*NettingResult {
	results := make(map[string]map[string]map[string]*NettingResult)

	getOrCreate := func(userID, symbol, currency string) *NettingResult {
		if _, ok := results[userID]; !ok {
			results[userID] = make(map[string]map[string]*NettingResult)
		}
		if _, ok := results[userID][symbol]; !ok {
			results[userID][symbol] = make(map[string]*NettingResult)
		}
		if _, ok := results[userID][symbol][currency]; !ok {
			results[userID][symbol][currency] = &NettingResult{
				UserID:   userID,
				Symbol:   symbol,
				Currency: currency,
			}
		}
		return results[userID][symbol][currency]
	}

	for _, st := range settlements {
		amount := st.TotalAmount
		quantity := st.Quantity

		buySide := getOrCreate(st.BuyUserID, st.Symbol, st.Currency)
		buySide.GrossBuy = buySide.GrossBuy.Add(amount)
		buySide.NetPosition = buySide.NetPosition.Add(quantity)
		buySide.NetAmount = buySide.NetAmount.Sub(amount)

		sellSide := getOrCreate(st.SellUserID, st.Symbol, st.Currency)
		sellSide.GrossSell = sellSide.GrossSell.Add(amount)
		sellSide.NetPosition = sellSide.NetPosition.Sub(quantity)
		sellSide.NetAmount = sellSide.NetAmount.Add(amount)
	}

	return results
}
