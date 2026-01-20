package mysql

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
	"gorm.io/gorm"
)

// StrategyPO
type StrategyPO struct {
	ID           uint            `gorm:"primarykey"`
	Symbol       string          `gorm:"column:symbol;type:varchar(20);uniqueIndex;not null"`
	Spread       decimal.Decimal `gorm:"column:spread;type:decimal(10,4);not null"`
	MinOrderSize decimal.Decimal `gorm:"column:min_order_size;type:decimal(32,18);not null"`
	MaxOrderSize decimal.Decimal `gorm:"column:max_order_size;type:decimal(32,18);not null"`
	MaxPosition  decimal.Decimal `gorm:"column:max_position;type:decimal(32,18);not null"`
	Status       string          `gorm:"column:status;type:varchar(20);not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (StrategyPO) TableName() string { return "marketmaking_strategies" }

func (po *StrategyPO) ToDomain() *domain.QuoteStrategy {
	return &domain.QuoteStrategy{
		Model:        gorm.Model{CreatedAt: po.CreatedAt, UpdatedAt: po.UpdatedAt, ID: po.ID},
		Symbol:       po.Symbol,
		Spread:       po.Spread,
		MinOrderSize: po.MinOrderSize,
		MaxOrderSize: po.MaxOrderSize,
		MaxPosition:  po.MaxPosition,
		Status:       domain.StrategyStatus(po.Status),
	}
}

func (po *StrategyPO) FromDomain(s *domain.QuoteStrategy) {
	po.Symbol = s.Symbol
	po.Spread = s.Spread
	po.MinOrderSize = s.MinOrderSize
	po.MaxOrderSize = s.MaxOrderSize
	po.MaxPosition = s.MaxPosition
	po.Status = string(s.Status)
	// CreatedAt/UpdatedAt handled by GORM on create/update usually, but can set manually
}
