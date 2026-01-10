// Package mysql 提供了做市合规策略与业绩仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// QuoteStrategyModel 策略数据库模型
type QuoteStrategyModel struct {
	gorm.Model
	Symbol       string `gorm:"column:symbol;type:varchar(20);uniqueIndex;not null"`
	Spread       string `gorm:"column:spread;type:decimal(32,18);not null"`
	MinOrderSize string `gorm:"column:min_order_size;type:decimal(32,18);not null"`
	MaxOrderSize string `gorm:"column:max_order_size;type:decimal(32,18);not null"`
	MaxPosition  string `gorm:"column:max_position;type:decimal(32,18);not null"`
	Status       string `gorm:"column:status;type:varchar(20);not null"`
}

func (QuoteStrategyModel) TableName() string { return "market_making_strategies" }

// PerformanceModel 绩效数据库模型
type PerformanceModel struct {
	gorm.Model
	Symbol      string    `gorm:"column:symbol;type:varchar(20);uniqueIndex;not null"`
	TotalPnL    string    `gorm:"column:total_pnl;type:decimal(32,18);default:0"`
	TotalVolume string    `gorm:"column:total_volume;type:decimal(32,18);default:0"`
	TotalTrades int64     `gorm:"column:total_trades;type:bigint;default:0"`
	SharpeRatio string    `gorm:"column:sharpe_ratio;type:decimal(32,18);default:0"`
	StartTime   time.Time `gorm:"column:start_time;type:datetime"`
	EndTime     time.Time `gorm:"column:end_time;type:datetime"`
}

func (PerformanceModel) TableName() string { return "market_making_performances" }

// implementation of QuoteStrategyRepository and PerformanceRepository
type marketMakingRepositoryImpl struct {
	db *gorm.DB
}

func NewMarketMakingRepository(db *gorm.DB) (domain.QuoteStrategyRepository, domain.PerformanceRepository) {
	impl := &marketMakingRepositoryImpl{db: db}
	return impl, impl
}

// QuoteStrategyRepository methods
func (r *marketMakingRepositoryImpl) SaveStrategy(ctx context.Context, strategy *domain.QuoteStrategy) error {
	m := &QuoteStrategyModel{
		Model:        strategy.Model,
		Symbol:       strategy.Symbol,
		Spread:       strategy.Spread.String(),
		MinOrderSize: strategy.MinOrderSize.String(),
		MaxOrderSize: strategy.MaxOrderSize.String(),
		MaxPosition:  strategy.MaxPosition.String(),
		Status:       string(strategy.Status),
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "symbol"}},
		UpdateAll: true,
	}).Create(m).Error
	if err == nil {
		strategy.Model = m.Model
	}
	return err
}

func (r *marketMakingRepositoryImpl) GetStrategyBySymbol(ctx context.Context, symbol string) (*domain.QuoteStrategy, error) {
	var m QuoteStrategyModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.strategyToDomain(&m), nil
}

// PerformanceRepository methods
func (r *marketMakingRepositoryImpl) SavePerformance(ctx context.Context, p *domain.MarketMakingPerformance) error {
	m := &PerformanceModel{
		Model:       p.Model,
		Symbol:      p.Symbol,
		TotalPnL:    p.TotalPnL.String(),
		TotalVolume: p.TotalVolume.String(),
		TotalTrades: p.TotalTrades,
		SharpeRatio: p.SharpeRatio.String(),
		StartTime:   p.StartTime,
		EndTime:     p.EndTime,
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "symbol"}},
		UpdateAll: true,
	}).Create(m).Error
	if err == nil {
		p.Model = m.Model
	}
	return err
}

func (r *marketMakingRepositoryImpl) GetPerformanceBySymbol(ctx context.Context, symbol string) (*domain.MarketMakingPerformance, error) {
	var m PerformanceModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.performanceToDomain(&m), nil
}

func (r *marketMakingRepositoryImpl) performanceToDomain(m *PerformanceModel) *domain.MarketMakingPerformance {
	pnl, err := decimal.NewFromString(m.TotalPnL)
	if err != nil {
		pnl = decimal.Zero
	}
	vol, err := decimal.NewFromString(m.TotalVolume)
	if err != nil {
		vol = decimal.Zero
	}
	sharpe, err := decimal.NewFromString(m.SharpeRatio)
	if err != nil {
		sharpe = decimal.Zero
	}
	return &domain.MarketMakingPerformance{
		Model:       m.Model,
		Symbol:      m.Symbol,
		TotalPnL:    pnl,
		TotalVolume: vol,
		TotalTrades: m.TotalTrades,
		SharpeRatio: sharpe,
		StartTime:   m.StartTime,
		EndTime:     m.EndTime,
	}
}

func (r *marketMakingRepositoryImpl) strategyToDomain(m *QuoteStrategyModel) *domain.QuoteStrategy {
	spread, err := decimal.NewFromString(m.Spread)
	if err != nil {
		spread = decimal.Zero
	}
	minSize, err := decimal.NewFromString(m.MinOrderSize)
	if err != nil {
		minSize = decimal.Zero
	}
	maxSize, err := decimal.NewFromString(m.MaxOrderSize)
	if err != nil {
		maxSize = decimal.Zero
	}
	pos, err := decimal.NewFromString(m.MaxPosition)
	if err != nil {
		pos = decimal.Zero
	}
	return &domain.QuoteStrategy{
		Model:        m.Model,
		Symbol:       m.Symbol,
		Spread:       spread,
		MinOrderSize: minSize,
		MaxOrderSize: maxSize,
		MaxPosition:  pos,
		Status:       domain.StrategyStatus(m.Status),
	}
}
