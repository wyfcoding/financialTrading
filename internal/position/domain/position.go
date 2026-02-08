package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// CostBasisMethod 成本计算方法
type CostBasisMethod string

const (
	CostBasisFIFO    CostBasisMethod = "FIFO"
	CostBasisLIFO    CostBasisMethod = "LIFO"
	CostBasisAverage CostBasisMethod = "AVERAGE"
)

// PositionLot 仓位头寸记录 (用于 FIFO/LIFO)
type PositionLot struct {
	ID         uint            `json:"id"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	PositionID uint            `json:"position_id"`
	Quantity   decimal.Decimal `json:"quantity"`
	Price      decimal.Decimal `json:"price"`
}

// Position represents a user's holding in a symbol
type Position struct {
	ID                uint            `json:"id"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	UserID            string          `json:"user_id"`
	Symbol            string          `json:"symbol"`
	Quantity          decimal.Decimal `json:"quantity"`
	AverageEntryPrice decimal.Decimal `json:"average_entry_price"`
	RealizedPnL       decimal.Decimal `json:"realized_pnl"`
	UnrealizedPnL     decimal.Decimal `json:"unrealized_pnl"`
	MarginRequirement decimal.Decimal `json:"margin_requirement"`
	Method            CostBasisMethod `json:"method"`
	Lots              []PositionLot   `json:"lots,omitempty"`
}

func NewPosition(userID, symbol string) *Position {
	return &Position{
		UserID:            userID,
		Symbol:            symbol,
		Quantity:          decimal.Zero,
		AverageEntryPrice: decimal.Zero,
		RealizedPnL:       decimal.Zero,
		UnrealizedPnL:     decimal.Zero,
		Method:            CostBasisAverage,
	}
}

// UpdatePosition updates the position based on a trade execution
func (p *Position) UpdatePosition(side string, qty, price decimal.Decimal) ([]PositionLot, []uint) {
	var created []PositionLot
	var deleted []uint

	isBuy := (side == "buy" || side == "BUY")

	// 1. 如果是增加现有头寸方向 (或开新仓)
	if (isBuy && !p.Quantity.IsNegative()) || (!isBuy && !p.Quantity.IsPositive()) {
		// 计算平均价 (始终维护平均价作为参考)
		absQty := p.Quantity.Abs()
		totalValue := absQty.Mul(p.AverageEntryPrice).Add(qty.Mul(price))

		change := qty
		if !isBuy {
			change = qty.Neg()
		}
		p.Quantity = p.Quantity.Add(change)

		if !p.Quantity.IsZero() {
			p.AverageEntryPrice = totalValue.Div(p.Quantity.Abs())
		}

		// 记录 Lot
		lot := PositionLot{
			PositionID: p.ID,
			Quantity:   qty,
			Price:      price,
		}
		p.Lots = append(p.Lots, lot)
		created = append(created, lot)
	} else {
		// 2. 减少现有头寸方向 (平仓/反手)
		remQty := qty
		for remQty.IsPositive() && len(p.Lots) > 0 {
			var idx int
			if p.Method == CostBasisLIFO {
				idx = len(p.Lots) - 1
			} else {
				idx = 0
			}

			lot := &p.Lots[idx]
			matchQty := decimal.Min(remQty, lot.Quantity)

			// 计算盈亏
			var pnl decimal.Decimal
			if isBuy { // 正在平空头
				pnl = lot.Price.Sub(price).Mul(matchQty)
			} else { // 正在平多头
				pnl = price.Sub(lot.Price).Mul(matchQty)
			}
			p.RealizedPnL = p.RealizedPnL.Add(pnl)

			remQty = remQty.Sub(matchQty)
			lot.Quantity = lot.Quantity.Sub(matchQty)

			change := matchQty
			if !isBuy {
				change = matchQty.Neg()
			}
			p.Quantity = p.Quantity.Add(change)

			if lot.Quantity.IsZero() {
				if lot.ID != 0 {
					deleted = append(deleted, lot.ID)
				}
				p.Lots = append(p.Lots[:idx], p.Lots[idx+1:]...)
			}
		}

		// 如果还有剩余 qty，说明发生了反手 (Flip)
		if remQty.IsPositive() {
			p.AverageEntryPrice = price
			if isBuy {
				p.Quantity = remQty
			} else {
				p.Quantity = remQty.Neg()
			}
			lot := PositionLot{
				PositionID: p.ID,
				Quantity:   remQty,
				Price:      price,
			}
			p.Lots = append(p.Lots, lot)
			created = append(created, lot)
		}

		if p.Quantity.IsZero() {
			p.AverageEntryPrice = decimal.Zero
		}
	}
	return created, deleted
}

// MarkToMarket 计算浮动盈亏
func (p *Position) MarkToMarket(currentPrice decimal.Decimal) {
	if p.Quantity.IsZero() {
		p.UnrealizedPnL = decimal.Zero
		return
	}

	// UnrealizedPnL = (CurrentPrice - AverageEntryPrice) * Quantity
	p.UnrealizedPnL = currentPrice.Sub(p.AverageEntryPrice).Mul(p.Quantity)
}
