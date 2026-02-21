package domain

import (
	"github.com/wyfcoding/pkg/money"
	"gorm.io/gorm"
)

type Portfolio struct {
	gorm.Model
	AccountID   string      `gorm:"column:account_id;uniqueIndex"`
	CashBalance money.Money `gorm:"embedded;embeddedPrefix:cash_"`
	TotalAsset  money.Money `gorm:"embedded;embeddedPrefix:total_"`
}
