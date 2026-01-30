package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
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
	gorm.Model
	SettlementID string           `gorm:"column:settlement_id;type:varchar(32);uniqueIndex;not null;comment:结算ID"`
	TradeID      string           `gorm:"column:trade_id;type:varchar(32);index;not null;comment:成交ID"`
	BuyUserID    string           `gorm:"column:buy_user_id;type:varchar(32);not null;comment:买方用户ID"`
	SellUserID   string           `gorm:"column:sell_user_id;type:varchar(32);not null;comment:卖方用户ID"`
	Symbol       string           `gorm:"column:symbol;type:varchar(20);not null;comment:标的"`
	Quantity     decimal.Decimal  `gorm:"column:quantity;type:decimal(32,18);not null;comment:数量"`
	Price        decimal.Decimal  `gorm:"column:price;type:decimal(32,18);not null;comment:价格"`
	TotalAmount  decimal.Decimal  `gorm:"column:total_amount;type:decimal(32,18);not null;comment:总金额"`
	Fee          decimal.Decimal  `gorm:"column:fee;type:decimal(32,18);default:0;not null;comment:手续费"`
	Status       SettlementStatus `gorm:"column:status;type:varchar(20);not null;comment:状态"`
	SettledAt    *time.Time       `gorm:"column:settled_at;comment:结算时间"`
	ErrorMessage string           `gorm:"column:error_message;type:text;comment:错误信息"`
}

func (Settlement) TableName() string {
	return "settlements"
}

// NewSettlement 创建新的结算单
func NewSettlement(settlementID, tradeID, buyUser, sellUser, symbol string, qty, price decimal.Decimal) *Settlement {
	total := qty.Mul(price)
	return &Settlement{
		SettlementID: settlementID,
		TradeID:      tradeID,
		BuyUserID:    buyUser,
		SellUserID:   sellUser,
		Symbol:       symbol,
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
