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
	ID           uint             `json:"id"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	SettlementID string           `json:"settlement_id"`
	TradeID      string           `json:"trade_id"`
	BuyUserID    string           `json:"buy_user_id"`
	SellUserID   string           `json:"sell_user_id"`
	Symbol       string           `json:"symbol"`
	Currency     string           `json:"currency"`
	Quantity     decimal.Decimal  `json:"quantity"`
	Price        decimal.Decimal  `json:"price"`
	TotalAmount  decimal.Decimal  `json:"total_amount"`
	Fee          decimal.Decimal  `json:"fee"`
	Status       SettlementStatus `json:"status"`
	SettledAt    *time.Time       `json:"settled_at"`
	ErrorMessage string           `json:"error_message"`
}

// NewSettlement 创建新的结算单
func NewSettlement(settlementID, tradeID, buyUser, sellUser, symbol, currency string, qty, price decimal.Decimal) *Settlement {
	total := qty.Mul(price)
	return &Settlement{
		SettlementID: settlementID,
		TradeID:      tradeID,
		BuyUserID:    buyUser,
		SellUserID:   sellUser,
		Symbol:       symbol,
		Currency:     currency,
		Quantity:     qty,
		Price:        price,
		TotalAmount:  total,
		Status:       StatusPending,
	}
}

// Complete 标记结算完成
func (s *Settlement) Complete() {
	now := time.Now()
	s.Status = StatusCompleted
	s.SettledAt = &now
}

// Fail 标记结算失败
func (s *Settlement) Fail(reason string) {
	s.Status = StatusFailed
	s.ErrorMessage = reason
}
