package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type PortfolioSnapshot struct {
	gorm.Model
	UserID        string          `gorm:"column:user_id;type:varchar(32);index:idx_user_snapshot,priority:1;not null"`
	SnapshotDate  time.Time       `gorm:"column:snapshot_date;type:date;index:idx_user_snapshot,priority:2;not null"`
	TotalEquity   decimal.Decimal `gorm:"column:total_equity;type:decimal(20,6);not null"`
	UnrealizedPnL decimal.Decimal `gorm:"column:unrealized_pnl;type:decimal(20,6);default:0"`
	RealizedPnL   decimal.Decimal `gorm:"column:realized_pnl;type:decimal(20,6);default:0"`
	DailyPnLPct   decimal.Decimal `gorm:"column:daily_pnl_pct;type:decimal(20,6);default:0"`
	Currency      string          `gorm:"column:currency;type:varchar(16);default:'USD'"`
}

func (PortfolioSnapshot) TableName() string { return "portfolio_snapshots" }

type UserPerformance struct {
	gorm.Model
	UserID           string          `gorm:"column:user_id;type:varchar(32);uniqueIndex;not null"`
	TotalReturn      decimal.Decimal `gorm:"column:total_return;type:decimal(20,6);default:0"`
	AnnualizedReturn decimal.Decimal `gorm:"column:annualized_return;type:decimal(20,6);default:0"`
	SharpeRatio      decimal.Decimal `gorm:"column:sharpe_ratio;type:decimal(20,6);default:0"`
	MaxDrawdown      decimal.Decimal `gorm:"column:max_drawdown;type:decimal(20,6);default:0"`
	WinRate          decimal.Decimal `gorm:"column:win_rate;type:decimal(20,6);default:0"`
}

func (UserPerformance) TableName() string { return "user_performance" }
