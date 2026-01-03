// 包 domain 持仓服务的领域模型
package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Position 持仓实体
// 代表用户在某个交易对上的持仓信息
type Position struct {
	gorm.Model
	// 持仓 ID
	PositionID string `gorm:"column:position_id;type:varchar(32);uniqueIndex;not null" json:"position_id"`
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(32);index;not null" json:"user_id"`
	// 交易对
	Symbol string `gorm:"column:symbol;type:varchar(20);index;not null" json:"symbol"`
	// 买卖方向 (LONG/SHORT)
	Side string `gorm:"column:side;type:varchar(10);not null" json:"side"`
	// 持仓数量
	Quantity decimal.Decimal `gorm:"column:quantity;type:decimal(32,18);not null" json:"quantity"`
	// 开仓价格 (平均成本)
	EntryPrice decimal.Decimal `gorm:"column:entry_price;type:decimal(32,18);not null" json:"entry_price"`
	// 当前价格
	CurrentPrice decimal.Decimal `gorm:"column:current_price;type:decimal(32,18);not null" json:"current_price"`
	// 未实现盈亏
	UnrealizedPnL decimal.Decimal `gorm:"column:unrealized_pnl;type:decimal(32,18);not null" json:"unrealized_pnl"`
	// 已实现盈亏
	RealizedPnL decimal.Decimal `gorm:"column:realized_pnl;type:decimal(32,18);not null" json:"realized_pnl"`
	// 开仓时间
	OpenedAt time.Time `gorm:"column:opened_at;type:datetime;not null" json:"opened_at"`
	// 平仓时间
	ClosedAt *time.Time `gorm:"column:closed_at;type:datetime" json:"closed_at"`
	// 状态
	Status string `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
}

// AddQuantity 增加持仓数量并滚动计算开仓均价 (加权平均成本法)
func (p *Position) AddQuantity(qty, price decimal.Decimal) {
	if qty.IsZero() {
		return
	}

	// 新开仓均价 = (旧持仓量 * 旧均价 + 新成交量 * 成交价) / (旧持仓量 + 新成交量)
	totalCost := p.Quantity.Mul(p.EntryPrice).Add(qty.Mul(price))
	newQty := p.Quantity.Add(qty)
	
	if !newQty.IsZero() {
		p.EntryPrice = totalCost.Div(newQty)
	}
	p.Quantity = newQty
}

// End of domain file
