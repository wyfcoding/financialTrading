package domain

import (
	"errors"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type LoanStatus int8

const (
	LoanPending    LoanStatus = 1
	LoanActive     LoanStatus = 2
	LoanRepaid     LoanStatus = 3
	LoanLiquidated LoanStatus = 4
)

// MarginLoan 融资融券借贷记录
type MarginLoan struct {
	gorm.Model
	LoanID    string          `gorm:"column:loan_id;type:varchar(32);unique_index;not null"`
	UserID    string          `gorm:"column:user_id;type:varchar(32);index;not null"`
	Asset     string          `gorm:"column:asset;type:varchar(20);not null"`
	Principal decimal.Decimal `gorm:"column:principal;type:decimal(32,16);not null"`
	Interest  decimal.Decimal `gorm:"column:interest;type:decimal(32,16);not null"`
	Rate      decimal.Decimal `gorm:"column:rate;type:decimal(16,8);not null"`
	Status    LoanStatus      `gorm:"column:status;type:tinyint;not null;default:1"`
}

func (MarginLoan) TableName() string { return "margin_loans" }

func NewMarginLoan(id, userID, asset string, amount decimal.Decimal, rate decimal.Decimal) *MarginLoan {
	return &MarginLoan{
		LoanID:    id,
		UserID:    userID,
		Asset:     asset,
		Principal: amount,
		Interest:  decimal.Zero,
		Rate:      rate,
		Status:    LoanActive,
	}
}

func (l *MarginLoan) Repay(amount decimal.Decimal) error {
	if l.Status != LoanActive {
		return errors.New("loan not active")
	}
	total := l.Principal.Add(l.Interest)
	if amount.GreaterThan(total) {
		return errors.New("repayment amount exceeds total debt")
	}

	// 简单逻辑：先扣利息，再扣本金
	if amount.GreaterThanOrEqual(l.Interest) {
		amount = amount.Sub(l.Interest)
		l.Interest = decimal.Zero
		l.Principal = l.Principal.Sub(amount)
	} else {
		l.Interest = l.Interest.Sub(amount)
	}

	if l.Principal.IsZero() {
		l.Status = LoanRepaid
	}
	return nil
}
