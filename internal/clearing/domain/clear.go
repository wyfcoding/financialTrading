//go:build clearing_legacy_fee
// +build clearing_legacy_fee

// 变更说明：
// 清算服务 (Clearing) 收编费率管理 (FeeManagement)。
package domain

import (
	"context"
	"github.com/shopspring/decimal"
)

// ClearingTrade 由撮合引擎生成，推入清算进行对账与算费的原始凭证。
type ClearingTrade struct {
	TradeID      uint64
	MakerAccount uint64
	TakerAccount uint64
	Symbol       string
	Price        decimal.Decimal
	Quantity     decimal.Decimal
	IsMakerBuy   bool
}

// FeeSchedule 费率表。收编自原始散乱出来的 FeeManagement 服务。
type FeeSchedule struct {
	TierID       int
	MakerFeeRate decimal.Decimal
	TakerFeeRate decimal.Decimal
}

// ClearingResult 清算结果，包含最终该收谁多少钱。
type ClearingResult struct {
	TradeID     uint64
	MakerFee    decimal.Decimal
	TakerFee    decimal.Decimal
	MakerPayout decimal.Decimal // Maker 最终应得或应付
	TakerPayout decimal.Decimal // Taker 最终应得或应付
}

// ClearTrade 计算费用及最终双边资金流转头寸。
func ClearTrade(trade *ClearingTrade, makerFeeRate, takerFeeRate decimal.Decimal) *ClearingResult {
	notional := trade.Price.Mul(trade.Quantity) // 交易本金金额

	makerFee := notional.Mul(makerFeeRate)
	takerFee := notional.Mul(takerFeeRate)

	result := &ClearingResult{
		TradeID:  trade.TradeID,
		MakerFee: makerFee,
		TakerFee: takerFee,
	}

	if trade.IsMakerBuy {
		// Maker 买，Taker 卖
		// 实际上 Maker 需要付出 本金+费，Taker 得到 本金-费
		result.MakerPayout = notional.Add(makerFee).Neg() // 负数代表扣款
		result.TakerPayout = notional.Sub(takerFee)       // 正数代表进账
	} else {
		// Maker 卖，Taker 买
		result.MakerPayout = notional.Sub(makerFee)
		result.TakerPayout = notional.Add(takerFee).Neg()
	}

	return result
}

type ClearingRepository interface {
	GetFeeSchedule(ctx context.Context, accountID uint64) (*FeeSchedule, error)
	SaveClearingResult(ctx context.Context, result *ClearingResult) error
}
