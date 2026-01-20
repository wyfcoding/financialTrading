package mysql

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	"gorm.io/gorm"
)

// SettlementPO 结算单持久化对象
type SettlementPO struct {
	gorm.Model
	SettlementID string          `gorm:"column:settlement_id;type:varchar(32);uniqueIndex;not null"`
	TradeID      string          `gorm:"column:trade_id;type:varchar(32);index;not null"`
	BuyUserID    string          `gorm:"column:buy_user_id;type:varchar(32);not null"`
	SellUserID   string          `gorm:"column:sell_user_id;type:varchar(32);not null"`
	Symbol       string          `gorm:"column:symbol;type:varchar(20);not null"`
	Quantity     decimal.Decimal `gorm:"column:quantity;type:decimal(32,18);not null"`
	Price        decimal.Decimal `gorm:"column:price;type:decimal(32,18);not null"`
	TotalAmount  decimal.Decimal `gorm:"column:total_amount;type:decimal(32,18);not null"`
	Status       string          `gorm:"column:status;type:varchar(20);not null"`
	SettledAt    *time.Time      `gorm:"column:settled_at"`
	ErrorMessage string          `gorm:"column:error_message;type:text"`
}

func (SettlementPO) TableName() string {
	return "settlements"
}

func (po *SettlementPO) ToDomain() *domain.Settlement {
	return &domain.Settlement{
		ID:           po.SettlementID,
		TradeID:      po.TradeID,
		BuyUserID:    po.BuyUserID,
		SellUserID:   po.SellUserID,
		Symbol:       po.Symbol,
		Quantity:     po.Quantity,
		Price:        po.Price,
		TotalAmount:  po.TotalAmount,
		Status:       domain.SettlementStatus(po.Status),
		SettledAt:    po.SettledAt,
		ErrorMessage: po.ErrorMessage,
		CreatedAt:    po.CreatedAt,
		UpdatedAt:    po.UpdatedAt,
	}
}

func (po *SettlementPO) FromDomain(s *domain.Settlement) {
	po.SettlementID = s.ID
	po.TradeID = s.TradeID
	po.BuyUserID = s.BuyUserID
	po.SellUserID = s.SellUserID
	po.Symbol = s.Symbol
	po.Quantity = s.Quantity
	po.Price = s.Price
	po.TotalAmount = s.TotalAmount
	po.Status = string(s.Status)
	po.SettledAt = s.SettledAt
	po.ErrorMessage = s.ErrorMessage
}
