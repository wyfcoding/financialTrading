// 变更说明：新增中央对手方 (CCP) 清算模型，实现“合同替代 (Novation)”逻辑，由清算所承担履约担保。
// 假设：清算所作为交易双方的共同对手方，每笔交易生成的 Settlement 将被拆分为清算所与买方、清算所与卖方的两条结算路径。
package domain

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// CCP Clearing House ID
const ClearingHouseUserID = "CLEARING_HOUSE"

// CCPContract 中央对手方合同 (Novated Contract)
type CCPContract struct {
	ContractID   string
	OriginalID   string
	Counterparty string
	Symbol       string
	Side         string // BUY/SELL (相对于对手方)
	Quantity     decimal.Decimal
	Price        decimal.Decimal
	TotalAmount  decimal.Decimal
	Status       string // OPEN, SETTLED, DEFAULTED
	CreatedAt    time.Time
}

// ClearingHouse 中央对手方服务
type ClearingHouse struct {
	Symbol string
}

func NewClearingHouse(symbol string) *ClearingHouse {
	return &ClearingHouse{Symbol: symbol}
}

// NovateContracts 执行合同替代
// 将原始买卖双方的 Settlement 转换为清算所与各自的合同
func (c *ClearingHouse) NovateContracts(s *Settlement) ([]*CCPContract, error) {
	if s.Status != StatusPending {
		return nil, fmt.Errorf("settlement %s is not pending", s.SettlementID)
	}

	timestamp := time.Now().UnixNano()

	// 1. 合同一：清算所 (卖方) vs 原买方 (买方)
	c1 := &CCPContract{
		ContractID:   fmt.Sprintf("CCP-%s-B-%d", s.SettlementID, timestamp),
		OriginalID:   s.SettlementID,
		Counterparty: s.BuyUserID,
		Symbol:       s.Symbol,
		Side:         "BUY",
		Quantity:     s.Quantity,
		Price:        s.Price,
		TotalAmount:  s.TotalAmount,
		Status:       "OPEN",
		CreatedAt:    time.Now(),
	}

	// 2. 合同二：清算所 (买方) vs 原卖方 (卖方)
	c2 := &CCPContract{
		ContractID:   fmt.Sprintf("CCP-%s-S-%d", s.SettlementID, timestamp),
		OriginalID:   s.SettlementID,
		Counterparty: s.SellUserID,
		Symbol:       s.Symbol,
		Side:         "SELL",
		Quantity:     s.Quantity,
		Price:        s.Price,
		TotalAmount:  s.TotalAmount,
		Status:       "OPEN",
		CreatedAt:    time.Now(),
	}

	return []*CCPContract{c1, c2}, nil
}
