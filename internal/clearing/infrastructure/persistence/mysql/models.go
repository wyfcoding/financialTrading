package mysql

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
)

// SettlementModel MySQL 结算表映射
type SettlementModel struct {
	ID           uint            `gorm:"primaryKey;autoIncrement"`
	CreatedAt    time.Time       `gorm:"column:created_at"`
	UpdatedAt    time.Time       `gorm:"column:updated_at"`
	SettlementID string          `gorm:"column:settlement_id;type:varchar(32);uniqueIndex;not null;comment:结算ID"`
	TradeID      string          `gorm:"column:trade_id;type:varchar(32);index;not null;comment:成交ID"`
	BuyUserID    string          `gorm:"column:buy_user_id;type:varchar(32);not null;comment:买方用户ID"`
	SellUserID   string          `gorm:"column:sell_user_id;type:varchar(32);not null;comment:卖方用户ID"`
	Symbol       string          `gorm:"column:symbol;type:varchar(20);not null;comment:标的"`
	Quantity     decimal.Decimal `gorm:"column:quantity;type:decimal(32,18);not null;comment:数量"`
	Price        decimal.Decimal `gorm:"column:price;type:decimal(32,18);not null;comment:价格"`
	TotalAmount  decimal.Decimal `gorm:"column:total_amount;type:decimal(32,18);not null;comment:总金额"`
	Fee          decimal.Decimal `gorm:"column:fee;type:decimal(32,18);default:0;not null;comment:手续费"`
	Status       string          `gorm:"column:status;type:varchar(20);not null;comment:状态"`
	SettledAt    *time.Time      `gorm:"column:settled_at;comment:结算时间"`
	ErrorMessage string          `gorm:"column:error_message;type:text;comment:错误信息"`
}

func (SettlementModel) TableName() string {
	return "settlements"
}

func toSettlementModel(s *domain.Settlement) *SettlementModel {
	if s == nil {
		return nil
	}
	return &SettlementModel{
		ID:           s.ID,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
		SettlementID: s.SettlementID,
		TradeID:      s.TradeID,
		BuyUserID:    s.BuyUserID,
		SellUserID:   s.SellUserID,
		Symbol:       s.Symbol,
		Quantity:     s.Quantity,
		Price:        s.Price,
		TotalAmount:  s.TotalAmount,
		Fee:          s.Fee,
		Status:       string(s.Status),
		SettledAt:    s.SettledAt,
		ErrorMessage: s.ErrorMessage,
	}
}

func toSettlement(model *SettlementModel) *domain.Settlement {
	if model == nil {
		return nil
	}
	return &domain.Settlement{
		ID:           model.ID,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
		SettlementID: model.SettlementID,
		TradeID:      model.TradeID,
		BuyUserID:    model.BuyUserID,
		SellUserID:   model.SellUserID,
		Symbol:       model.Symbol,
		Quantity:     model.Quantity,
		Price:        model.Price,
		TotalAmount:  model.TotalAmount,
		Fee:          model.Fee,
		Status:       domain.SettlementStatus(model.Status),
		SettledAt:    model.SettledAt,
		ErrorMessage: model.ErrorMessage,
	}
}
