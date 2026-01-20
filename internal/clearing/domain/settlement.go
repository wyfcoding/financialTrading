package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// SettlementStatus 结算状态
type SettlementStatus string

const (
	StatusPending   SettlementStatus = "PENDING"
	StatusCompleted SettlementStatus = "COMPLETED"
	StatusFailed    SettlementStatus = "FAILED"
)

// Settlement 结算单聚合根
type Settlement struct {
	ID           string
	TradeID      string
	BuyUserID    string
	SellUserID   string
	Symbol       string
	Quantity     decimal.Decimal
	Price        decimal.Decimal
	TotalAmount  decimal.Decimal
	Fee          decimal.Decimal
	Status       SettlementStatus
	SettledAt    *time.Time
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewSettlement 创建新的结算单
func NewSettlement(id, tradeID, buyUser, sellUser, symbol string, qty, price decimal.Decimal) *Settlement {
	total := qty.Mul(price)
	return &Settlement{
		ID:          id,
		TradeID:     tradeID,
		BuyUserID:   buyUser,
		SellUserID:  sellUser,
		Symbol:      symbol,
		Quantity:    qty,
		Price:       price,
		TotalAmount: total,
		Status:      StatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// Complete 标记结算完成
func (s *Settlement) Complete() {
	now := time.Now()
	s.Status = StatusCompleted
	s.SettledAt = &now
	s.UpdatedAt = now
}

// Fail 标记结算失败
func (s *Settlement) Fail(reason string) {
	s.Status = StatusFailed
	s.ErrorMessage = reason
	s.UpdatedAt = time.Now()
}
