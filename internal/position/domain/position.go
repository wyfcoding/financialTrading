package domain

import (
	"gorm.io/gorm"
)

// Position represents a user's holding in a symbol
type Position struct {
	gorm.Model
	UserID            string  `gorm:"column:user_id;type:varchar(50);index;uniqueIndex:idx_user_symbol"`
	Symbol            string  `gorm:"column:symbol;type:varchar(20);index;uniqueIndex:idx_user_symbol"`
	Quantity          float64 `gorm:"column:quantity;type:decimal(20,8)"`
	AverageEntryPrice float64 `gorm:"column:average_entry_price;type:decimal(20,8)"`
	RealizedPnL       float64 `gorm:"column:realized_pnl;type:decimal(20,8);default:0"`
}

func (p *Position) TableName() string {
	return "positions"
}

func NewPosition(userID, symbol string) *Position {
	return &Position{
		UserID:            userID,
		Symbol:            symbol,
		Quantity:          0,
		AverageEntryPrice: 0,
		RealizedPnL:       0,
	}
}

// UpdatePosition updates the position based on a trade execution
// This is a simplified Average Cost Basis implementation
func (p *Position) UpdatePosition(side string, qty, price float64) {
	if side == "buy" {
		if p.Quantity >= 0 {
			// Long + Buy -> Add to position, update avg price
			totalValue := (p.Quantity * p.AverageEntryPrice) + (qty * price)
			p.Quantity += qty
			p.AverageEntryPrice = totalValue / p.Quantity
		} else {
			// Short + Buy -> Reducing position (Covering)
			// e.g. Short 10 @ 100. Buy 5 @ 90.
			// Realized PnL on 5 units = (Entry 100 - Exit 90) * 5 = 50
			// Remaining Short 5 @ 100.

			// If covering more than open, flip to Long
			remainingQty := p.Quantity + qty

			if remainingQty > 0 {
				// Flip short to long
				// 1. Cover entire short
				coverQty := -p.Quantity
				pnl := (p.AverageEntryPrice - price) * coverQty
				p.RealizedPnL += pnl

				// 2. Open new long
				newLongQty := remainingQty
				p.Quantity = newLongQty
				p.AverageEntryPrice = price
			} else {
				// Reduce short
				// qty is positive (buy), p.Quantity is negative
				// covered amount is `qty`
				pnl := (p.AverageEntryPrice - price) * qty
				p.RealizedPnL += pnl
				p.Quantity += qty // approaches 0
				if p.Quantity == 0 {
					p.AverageEntryPrice = 0
				}
			}
		}
	} else { // SELL
		if p.Quantity <= 0 {
			// Short + Sell -> Add to short position, update avg price
			// Use abs values for calculation logic
			absQty := -p.Quantity
			totalValue := (absQty * p.AverageEntryPrice) + (qty * price)
			p.Quantity -= qty // becomes more negative
			p.AverageEntryPrice = totalValue / (-p.Quantity)
		} else {
			// Long + Sell -> Reducing position
			// e.g. Long 10 @ 100. Sell 5 @ 110.
			// PnL = (110 - 100) * 5 = 50.

			remainingQty := p.Quantity - qty

			if remainingQty < 0 {
				// Flip long to short
				// 1. Close entire long
				closeQty := p.Quantity
				pnl := (price - p.AverageEntryPrice) * closeQty
				p.RealizedPnL += pnl

				// 2. Open new short
				newShortQty := remainingQty // negative
				p.Quantity = newShortQty
				p.AverageEntryPrice = price
			} else {
				// Reduce long
				pnl := (price - p.AverageEntryPrice) * qty
				p.RealizedPnL += pnl
				p.Quantity -= qty
				if p.Quantity == 0 {
					p.AverageEntryPrice = 0
				}
			}
		}
	}
}
