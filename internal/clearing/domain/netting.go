package domain

import (
	"github.com/shopspring/decimal"
)

// NettingResult 净额清算结果
type NettingResult struct {
	UserID      string          `json:"user_id"`
	Symbol      string          `json:"symbol"`
	NetQuantity decimal.Decimal `json:"net_quantity"` // 正数表示应收(Long)，负数表示应付(Short)
	NetAmount   decimal.Decimal `json:"net_amount"`   // 正数表示应收(资金)，负数表示应付(资金)
	Currency    string          `json:"currency"`
}

// NettingEngine 净额清算引擎
type NettingEngine struct{}

// NewNettingEngine 创建净额清算引擎
func NewNettingEngine() *NettingEngine {
	return &NettingEngine{}
}

// CalculateMultilateralNetting 计算多边净额
// 输入：一批待清算的结算单
// 输出：每个用户在该批次中的净头寸变动和资金变动 (按币种分组)
func (e *NettingEngine) CalculateMultilateralNetting(settlements []*Settlement) map[string]map[string]map[string]*NettingResult {
	// 结果集: UserID -> Symbol -> Currency -> Result
	results := make(map[string]map[string]map[string]*NettingResult)

	for _, s := range settlements {
		if s.Status != StatusPending {
			continue
		}

		// 处理买方 (获得 Asset，支付 Cash)
		e.updateResult(results, s.BuyUserID, s.Symbol, s.Currency, s.Quantity, s.TotalAmount.Neg())

		// 处理卖方 (失去 Asset，获得 Cash)
		e.updateResult(results, s.SellUserID, s.Symbol, s.Currency, s.Quantity.Neg(), s.TotalAmount)
	}

	return results
}

func (e *NettingEngine) updateResult(results map[string]map[string]map[string]*NettingResult, userID, symbol, currency string, qtyDelta, amountDelta decimal.Decimal) {
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

	res := results[userID][symbol][currency]
	res.NetQuantity = res.NetQuantity.Add(qtyDelta)
	res.NetAmount = res.NetAmount.Add(amountDelta)
}
