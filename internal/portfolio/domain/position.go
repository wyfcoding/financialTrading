package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Position struct {
	gorm.Model
	UserID       string          `gorm:"column:user_id;type:varchar(32);index:idx_user_symbol;not null"`
	Symbol       string          `gorm:"column:symbol;type:varchar(20);index:idx_user_symbol;not null"`
	Quantity     decimal.Decimal `gorm:"column:quantity;type:decimal(20,6);not null"`
	AvailableQty decimal.Decimal `gorm:"column:available_qty;type:decimal(20,6);not null"`
	FrozenQty    decimal.Decimal `gorm:"column:frozen_qty;type:decimal(20,6);not null"`
	AvgCost      decimal.Decimal `gorm:"column:avg_cost;type:decimal(20,6);not null"`
	UnrealizedPnL decimal.Decimal `gorm:"column:unrealized_pnl;type:decimal(20,6)"`
	RealizedPnL  decimal.Decimal `gorm:"column:realized_pnl;type:decimal(20,6)"`
	PositionType string          `gorm:"column:position_type;type:varchar(20);not null"`
	Leverage     decimal.Decimal `gorm:"column:leverage;type:decimal(10,2);default:1"`
	MarginUsed   decimal.Decimal `gorm:"column:margin_used;type:decimal(20,6)"`
	LiquidationPrice decimal.Decimal `gorm:"column:liquidation_price;type:decimal(20,6)"`
	OpenedAt     time.Time       `gorm:"column:opened_at;type:timestamp"`
	UpdatedAt    time.Time       `gorm:"column:updated_at;type:timestamp"`
}

func (Position) TableName() string { return "positions" }

func NewPosition(userID, symbol string, qty, avgCost decimal.Decimal, posType string) *Position {
	now := time.Now()
	return &Position{
		UserID:       userID,
		Symbol:       symbol,
		Quantity:     qty,
		AvailableQty: qty,
		FrozenQty:    decimal.Zero,
		AvgCost:      avgCost,
		PositionType: posType,
		Leverage:     decimal.NewFromInt(1),
		OpenedAt:     now,
		UpdatedAt:    now,
	}
}

func (p *Position) AddQuantity(qty, price decimal.Decimal) {
	totalCost := p.AvgCost.Mul(p.Quantity).Add(price.Mul(qty))
	newQty := p.Quantity.Add(qty)
	if !newQty.IsZero() {
		p.AvgCost = totalCost.Div(newQty)
	}
	p.Quantity = newQty
	p.AvailableQty = p.AvailableQty.Add(qty)
	p.UpdatedAt = time.Now()
}

func (p *Position) ReduceQuantity(qty, price decimal.Decimal) decimal.Decimal {
	realizedPnL := price.Sub(p.AvgCost).Mul(qty)
	p.Quantity = p.Quantity.Sub(qty)
	p.AvailableQty = p.AvailableQty.Sub(qty)
	p.RealizedPnL = p.RealizedPnL.Add(realizedPnL)
	p.UpdatedAt = time.Now()
	return realizedPnL
}

func (p *Position) Freeze(qty decimal.Decimal) bool {
	if p.AvailableQty.LessThan(qty) {
		return false
	}
	p.AvailableQty = p.AvailableQty.Sub(qty)
	p.FrozenQty = p.FrozenQty.Add(qty)
	p.UpdatedAt = time.Now()
	return true
}

func (p *Position) Unfreeze(qty decimal.Decimal) bool {
	if p.FrozenQty.LessThan(qty) {
		return false
	}
	p.FrozenQty = p.FrozenQty.Sub(qty)
	p.AvailableQty = p.AvailableQty.Add(qty)
	p.UpdatedAt = time.Now()
	return true
}

func (p *Position) UpdateUnrealizedPnL(currentPrice decimal.Decimal) {
	if p.Quantity.IsZero() {
		p.UnrealizedPnL = decimal.Zero
		return
	}
	p.UnrealizedPnL = currentPrice.Sub(p.AvgCost).Mul(p.Quantity)
}

func (p *Position) UpdateLiquidationPrice(maintenanceMarginRate decimal.Decimal) {
	if p.Leverage.IsZero() || p.Leverage.Equal(decimal.NewFromInt(1)) {
		p.LiquidationPrice = decimal.Zero
		return
	}
	p.LiquidationPrice = p.AvgCost.Mul(p.Leverage.Sub(decimal.NewFromInt(1))).Div(p.Leverage)
}

func (p *Position) IsLong() bool {
	return p.Quantity.IsPositive()
}

func (p *Position) IsShort() bool {
	return p.Quantity.IsNegative()
}

func (p *Position) IsEmpty() bool {
	return p.Quantity.IsZero()
}

func (p *Position) MarketValue(currentPrice decimal.Decimal) decimal.Decimal {
	return p.Quantity.Mul(currentPrice)
}

func (p *Position) CostBasis() decimal.Decimal {
	return p.AvgCost.Mul(p.Quantity).Abs()
}

type PortfolioOverview struct {
	UserID         string
	TotalEquity    decimal.Decimal
	TotalCost      decimal.Decimal
	UnrealizedPnL  decimal.Decimal
	RealizedPnL    decimal.Decimal
	DailyPnL       decimal.Decimal
	DailyPnLPct    decimal.Decimal
	Positions      []*Position
	Currency       string
	LastUpdated    time.Time
}

func NewPortfolioOverview(userID, currency string) *PortfolioOverview {
	return &PortfolioOverview{
		UserID:      userID,
		Currency:    currency,
		Positions:   make([]*Position, 0),
		LastUpdated: time.Now(),
	}
}

func (o *PortfolioOverview) AddPosition(p *Position) {
	o.Positions = append(o.Positions, p)
	o.TotalCost = o.TotalCost.Add(p.CostBasis())
	o.UnrealizedPnL = o.UnrealizedPnL.Add(p.UnrealizedPnL)
	o.RealizedPnL = o.RealizedPnL.Add(p.RealizedPnL)
}

func (o *PortfolioOverview) CalculateTotalEquity() {
	o.TotalEquity = o.TotalCost.Add(o.UnrealizedPnL)
}

func (o *PortfolioOverview) UpdateDailyPnL(previousEquity decimal.Decimal) {
	if !previousEquity.IsZero() {
		o.DailyPnL = o.TotalEquity.Sub(previousEquity)
		o.DailyPnLPct = o.DailyPnL.Div(previousEquity)
	}
}

type PortfolioEvent struct {
	ID          string
	UserID      string
	EventType   string
	Symbol      string
	Quantity    decimal.Decimal
	Price       decimal.Decimal
	Timestamp   time.Time
	Description string
}

const (
	EventTypePositionOpen   = "position_open"
	EventTypePositionClose  = "position_close"
	EventTypePositionAdjust = "position_adjust"
	EventTypeDividend       = "dividend"
	EventTypeSplit          = "split"
)
