package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// PortfolioSnapshot 每日资产快照，用于业绩分析
type PortfolioSnapshot struct {
	gorm.Model
	UserID      string          `gorm:"column:user_id;type:varchar(32);index:idx_user_date;not null"`
	Date        time.Time       `gorm:"column:date;type:date;index:idx_user_date;not null"`
	TotalEquity decimal.Decimal `gorm:"column:total_equity;type:decimal(20,2);not null"`
	Currency    string          `gorm:"column:currency;type:varchar(3);not null"`
}

func (PortfolioSnapshot) TableName() string { return "portfolio_snapshots" }

func NewPortfolioSnapshot(userID string, date time.Time, equity decimal.Decimal, curr string) *PortfolioSnapshot {
	return &PortfolioSnapshot{
		UserID:      userID,
		Date:        date,
		TotalEquity: equity,
		Currency:    curr,
	}
}

// UserPerformance 用户业绩指标
type UserPerformance struct {
	UserID      string          `gorm:"primaryKey;column:user_id;type:varchar(32)"`
	TotalReturn decimal.Decimal `gorm:"column:total_return;type:decimal(10,4)"` // 0.1500 for 15%
	SharpeRatio decimal.Decimal `gorm:"column:sharpe_ratio;type:decimal(10,4)"`
	MaxDrawdown decimal.Decimal `gorm:"column:max_drawdown;type:decimal(10,4)"`
	UpdatedAt   time.Time
}

func (UserPerformance) TableName() string { return "user_performance" }
