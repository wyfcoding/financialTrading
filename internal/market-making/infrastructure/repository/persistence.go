// 包 基础设施层实现
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/wyfcoding/financialTrading/internal/market-making/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
)

// QuoteStrategyModel 报价策略数据库模型
// 对应数据库中的 quote_strategies 表
type QuoteStrategyModel struct {
	gorm.Model
	ID           string  `gorm:"column:id;type:varchar(36);primaryKey;comment:策略ID"`
	Symbol       string  `gorm:"column:symbol;type:varchar(20);uniqueIndex;not null;comment:交易对"`
	Spread       float64 `gorm:"column:spread;type:decimal(10,4);not null;comment:目标点差"`
	MinOrderSize float64 `gorm:"column:min_order_size;type:decimal(20,8);not null;comment:最小订单量"`
	MaxOrderSize float64 `gorm:"column:max_order_size;type:decimal(20,8);not null;comment:最大订单量"`
	MaxPosition  float64 `gorm:"column:max_position;type:decimal(20,8);not null;comment:最大持仓"`
	Status       string  `gorm:"column:status;type:varchar(20);default:'ACTIVE';comment:状态"`
}

// 指定表名
func (QuoteStrategyModel) TableName() string {
	return "quote_strategies"
}

// 将数据库模型转换为领域实体
func (m *QuoteStrategyModel) ToDomain() *domain.QuoteStrategy {
	return &domain.QuoteStrategy{
		Model:        m.Model,
		ID:           m.ID,
		Symbol:       m.Symbol,
		Spread:       m.Spread,
		MinOrderSize: m.MinOrderSize,
		MaxOrderSize: m.MaxOrderSize,
		MaxPosition:  m.MaxPosition,
		Status:       domain.StrategyStatus(m.Status),
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// QuoteStrategyRepositoryImpl 策略仓储实现
type QuoteStrategyRepositoryImpl struct {
	db *gorm.DB
}

// NewQuoteStrategyRepository 创建报价策略仓储实例
func NewQuoteStrategyRepository(db *gorm.DB) domain.QuoteStrategyRepository {
	return &QuoteStrategyRepositoryImpl{db: db}
}

// Save 保存报价策略
func (r *QuoteStrategyRepositoryImpl) Save(ctx context.Context, strategy *domain.QuoteStrategy) error {
	model := &QuoteStrategyModel{
		Model:        strategy.Model,
		ID:           strategy.ID,
		Symbol:       strategy.Symbol,
		Spread:       strategy.Spread,
		MinOrderSize: strategy.MinOrderSize,
		MaxOrderSize: strategy.MaxOrderSize,
		MaxPosition:  strategy.MaxPosition,
		Status:       string(strategy.Status),
	}
	// 更新或插入 (Upsert)
	if err := r.db.WithContext(ctx).Where(QuoteStrategyModel{Symbol: strategy.Symbol}).Assign(model).FirstOrCreate(model).Error; err != nil {
		logging.Error(ctx, "Failed to save quote strategy",
			"symbol", strategy.Symbol,
			"error", err,
		)
		return fmt.Errorf("failed to save quote strategy: %w", err)
	}

	strategy.Model = model.Model
	return nil
}

// GetBySymbol 根据交易对获取报价策略
func (r *QuoteStrategyRepositoryImpl) GetBySymbol(ctx context.Context, symbol string) (*domain.QuoteStrategy, error) {
	var model QuoteStrategyModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get quote strategy by symbol",
			"symbol", symbol,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get quote strategy by symbol: %w", err)
	}
	return model.ToDomain(), nil
}

// PerformanceModel 绩效数据库模型
type PerformanceModel struct {
	gorm.Model
	Symbol      string  `gorm:"column:symbol;type:varchar(20);uniqueIndex;not null;comment:交易对"`
	TotalPnL    float64 `gorm:"column:total_pnl;type:decimal(20,8);default:0;comment:总盈亏"`
	TotalVolume float64 `gorm:"column:total_volume;type:decimal(20,8);default:0;comment:总成交量"`
	TotalTrades int     `gorm:"column:total_trades;type:int;default:0;comment:总成交次数"`
	SharpeRatio float64 `gorm:"column:sharpe_ratio;type:decimal(10,4);default:0;comment:夏普比率"`
}

// 指定表名
func (PerformanceModel) TableName() string {
	return "market_making_performance"
}

// ToDomain 转换为领域实体
func (m *PerformanceModel) ToDomain() *domain.MarketMakingPerformance {
	return &domain.MarketMakingPerformance{
		Model:       m.Model,
		Symbol:      m.Symbol,
		TotalPnL:    m.TotalPnL,
		TotalVolume: m.TotalVolume,
		TotalTrades: m.TotalTrades,
		SharpeRatio: m.SharpeRatio,
		StartTime:   m.CreatedAt,
		EndTime:     m.UpdatedAt,
	}
}

// PerformanceRepositoryImpl 绩效仓储实现
type PerformanceRepositoryImpl struct {
	db *gorm.DB
}

// NewPerformanceRepository 创建绩效仓储实例
func NewPerformanceRepository(db *gorm.DB) domain.PerformanceRepository {
	return &PerformanceRepositoryImpl{db: db}
}

// Save 保存绩效
func (r *PerformanceRepositoryImpl) Save(ctx context.Context, performance *domain.MarketMakingPerformance) error {
	model := &PerformanceModel{
		Model:       performance.Model,
		Symbol:      performance.Symbol,
		TotalPnL:    performance.TotalPnL,
		TotalVolume: performance.TotalVolume,
		TotalTrades: performance.TotalTrades,
		SharpeRatio: performance.SharpeRatio,
	}
	if err := r.db.WithContext(ctx).Where(PerformanceModel{Symbol: performance.Symbol}).Assign(model).FirstOrCreate(model).Error; err != nil {
		logging.Error(ctx, "Failed to save performance",
			"symbol", performance.Symbol,
			"error", err,
		)
		return fmt.Errorf("failed to save performance: %w", err)
	}

	performance.Model = model.Model
	return nil
}

// GetBySymbol 根据交易对获取绩效
func (r *PerformanceRepositoryImpl) GetBySymbol(ctx context.Context, symbol string) (*domain.MarketMakingPerformance, error) {
	var model PerformanceModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get performance by symbol",
			"symbol", symbol,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get performance by symbol: %w", err)
	}
	return model.ToDomain(), nil
}
