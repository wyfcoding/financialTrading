package domain

import (
	"math"
	"time"
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
	ID         uint      `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	PositionID uint      `json:"position_id"`
	Quantity   float64   `json:"quantity"`
	Price      float64   `json:"price"`
}

// Position represents a user's holding in a symbol
type Position struct {
	ID                uint            `json:"id"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	UserID            string          `json:"user_id"`
	Symbol            string          `json:"symbol"`
	Quantity          float64         `json:"quantity"`
	AverageEntryPrice float64         `json:"average_entry_price"`
	RealizedPnL       float64         `json:"realized_pnl"`
	UnrealizedPnL     float64         `json:"unrealized_pnl"`
	MarginRequirement float64         `json:"margin_requirement"`
	Method            CostBasisMethod `json:"method"`
	Lots              []PositionLot   `json:"lots,omitempty"`
}

func NewPosition(userID, symbol string) *Position {
	return &Position{
		UserID:            userID,
		Symbol:            symbol,
		Quantity:          0,
		AverageEntryPrice: 0,
		RealizedPnL:       0,
		Method:            CostBasisAverage,
	}
}

// UpdatePosition updates the position based on a trade execution
func (p *Position) UpdatePosition(side string, qty, price float64) ([]PositionLot, []uint) {
	var created []PositionLot
	var deleted []uint

	isBuy := (side == "buy" || side == "BUY")

	// 1. 如果是增加现有头寸方向 (或开新仓)
	if (isBuy && p.Quantity >= 0) || (!isBuy && p.Quantity <= 0) {
		// 计算平均价 (始终维护平均价作为参考)
		absQty := math.Abs(p.Quantity)
		totalValue := (absQty * p.AverageEntryPrice) + (qty * price)
		p.Quantity = p.Quantity + (func() float64 {
			if isBuy {
				return qty
			}
			return -qty
		}())
		p.AverageEntryPrice = totalValue / math.Abs(p.Quantity)

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
		for remQty > 0 && len(p.Lots) > 0 {
			var idx int
			if p.Method == CostBasisLIFO {
				idx = len(p.Lots) - 1
			} else {
				idx = 0
			}

			lot := &p.Lots[idx]
			matchQty := math.Min(remQty, lot.Quantity)

			// 计算盈亏
			var pnl float64
			if isBuy { // 正在平空头
				pnl = (lot.Price - price) * matchQty
			} else { // 正在平多头
				pnl = (price - lot.Price) * matchQty
			}
			p.RealizedPnL += pnl

			remQty -= matchQty
			lot.Quantity -= matchQty
			p.Quantity += (func() float64 {
				if isBuy {
					return matchQty
				}
				return -matchQty
			}())

			if lot.Quantity <= 0 {
				if lot.ID != 0 {
					deleted = append(deleted, lot.ID)
				}
				p.Lots = append(p.Lots[:idx], p.Lots[idx+1:]...)
			}
		}

		// 如果还有剩余 qty，说明发生了反手 (Flip)
		if remQty > 0 {
			p.AverageEntryPrice = price
			p.Quantity = (func() float64 {
				if isBuy {
					return remQty
				}
				return -remQty
			}())
			lot := PositionLot{
				PositionID: p.ID,
				Quantity:   remQty,
				Price:      price,
			}
			p.Lots = append(p.Lots, lot)
			created = append(created, lot)
		}

		if p.Quantity == 0 {
			p.AverageEntryPrice = 0
		}
	}
	return created, deleted
}
