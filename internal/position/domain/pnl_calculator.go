// 包 持仓服务的领域模型
package domain

import (
	"github.com/shopspring/decimal"
)

// PnLCalculator PnL 计算器
// 提供盈亏计算和均价更新的领域服务
type PnLCalculator struct{}

// NewPnLCalculator 创建 PnL 计算器实例
func NewPnLCalculator() *PnLCalculator {
	return &PnLCalculator{}
}

// CalculateUnrealizedPnL 计算未实现盈亏
// position: 持仓对象
// currentPrice: 当前市场价格
// 返回: 未实现盈亏金额
func (c *PnLCalculator) CalculateUnrealizedPnL(position *Position, currentPrice decimal.Decimal) decimal.Decimal {
	if position.Quantity.IsZero() {
		return decimal.Zero
	}

	// 未实现盈亏 = (当前价格 - 开仓均价) * 持仓数量 * 方向系数
	// 方向系数: 多头为 1, 空头为 -1

	diff := currentPrice.Sub(position.EntryPrice)
	pnl := diff.Mul(position.Quantity)

	if position.Side == "SELL" {
		pnl = pnl.Neg()
	}

	return pnl
}

// CalculateRealizedPnL 计算已实现盈亏
// closePrice: 平仓价格
// closeQuantity: 平仓数量
// entryPrice: 开仓均价
// side: 持仓方向
func (c *PnLCalculator) CalculateRealizedPnL(closePrice, closeQuantity, entryPrice decimal.Decimal, side string) decimal.Decimal {
	diff := closePrice.Sub(entryPrice)
	pnl := diff.Mul(closeQuantity)

	if side == "SELL" {
		pnl = pnl.Neg()
	}

	return pnl
}

// UpdateAveragePrice 更新持仓均价 (加权平均)
// currentQty: 当前持仓数量
// currentAvgPrice: 当前持仓均价
// newQty: 新增数量
// newPrice: 新增价格
func (c *PnLCalculator) UpdateAveragePrice(currentQty, currentAvgPrice, newQty, newPrice decimal.Decimal) decimal.Decimal {
	totalQty := currentQty.Add(newQty)
	if totalQty.IsZero() {
		return decimal.Zero
	}

	totalCost := currentQty.Mul(currentAvgPrice).Add(newQty.Mul(newPrice))
	return totalCost.Div(totalQty)
}
