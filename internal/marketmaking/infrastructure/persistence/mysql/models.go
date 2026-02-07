package mysql

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
	"gorm.io/gorm"
)

// StrategyModel MySQL 做市策略表映射
type StrategyModel struct {
	gorm.Model
	Symbol       string          `gorm:"column:symbol;type:varchar(32);uniqueIndex;not null"`
	Spread       decimal.Decimal `gorm:"column:spread;type:decimal(32,18);not null"`
	MinOrderSize decimal.Decimal `gorm:"column:min_order_size;type:decimal(32,18);not null"`
	MaxOrderSize decimal.Decimal `gorm:"column:max_order_size;type:decimal(32,18);not null"`
	MaxPosition  decimal.Decimal `gorm:"column:max_position;type:decimal(32,18);not null"`
	Status       string          `gorm:"column:status;type:varchar(20);not null"`
}

func (StrategyModel) TableName() string { return "marketmaking_strategies" }

// PerformanceModel MySQL 做市绩效表映射
type PerformanceModel struct {
	gorm.Model
	Symbol      string          `gorm:"column:symbol;type:varchar(32);uniqueIndex;not null"`
	TotalPnL    decimal.Decimal `gorm:"column:total_pnl;type:decimal(32,18);not null"`
	TotalVolume decimal.Decimal `gorm:"column:total_volume;type:decimal(32,18);not null"`
	TotalTrades int64           `gorm:"column:total_trades;not null"`
	SharpeRatio decimal.Decimal `gorm:"column:sharpe_ratio;type:decimal(18,8);not null"`
	StartTime   time.Time       `gorm:"column:start_time;not null"`
	EndTime     time.Time       `gorm:"column:end_time;not null"`
}

func (PerformanceModel) TableName() string { return "marketmaking_performance" }

func toStrategyModel(s *domain.QuoteStrategy) *StrategyModel {
	if s == nil {
		return nil
	}
	return &StrategyModel{
		Model: gorm.Model{
			ID:        s.ID,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		},
		Symbol:       s.Symbol,
		Spread:       s.Spread,
		MinOrderSize: s.MinOrderSize,
		MaxOrderSize: s.MaxOrderSize,
		MaxPosition:  s.MaxPosition,
		Status:       string(s.Status),
	}
}

func toStrategy(m *StrategyModel) *domain.QuoteStrategy {
	if m == nil {
		return nil
	}
	return &domain.QuoteStrategy{
		ID:           m.ID,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		Symbol:       m.Symbol,
		Spread:       m.Spread,
		MinOrderSize: m.MinOrderSize,
		MaxOrderSize: m.MaxOrderSize,
		MaxPosition:  m.MaxPosition,
		Status:       domain.StrategyStatus(m.Status),
	}
}

func toPerformanceModel(p *domain.MarketMakingPerformance) *PerformanceModel {
	if p == nil {
		return nil
	}
	return &PerformanceModel{
		Model: gorm.Model{
			ID:        p.ID,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
		Symbol:      p.Symbol,
		TotalPnL:    p.TotalPnL,
		TotalVolume: p.TotalVolume,
		TotalTrades: p.TotalTrades,
		SharpeRatio: p.SharpeRatio,
		StartTime:   p.StartTime,
		EndTime:     p.EndTime,
	}
}

func toPerformance(m *PerformanceModel) *domain.MarketMakingPerformance {
	if m == nil {
		return nil
	}
	return &domain.MarketMakingPerformance{
		ID:          m.ID,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		Symbol:      m.Symbol,
		TotalPnL:    m.TotalPnL,
		TotalVolume: m.TotalVolume,
		TotalTrades: m.TotalTrades,
		SharpeRatio: m.SharpeRatio,
		StartTime:   m.StartTime,
		EndTime:     m.EndTime,
	}
}
