// Package infrastructure 算法交易基础设施层
package infrastructure

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/algotrading/domain"
	"gorm.io/gorm"
)

// GormStrategyRepository GORM 策略仓储
type GormStrategyRepository struct {
	db *gorm.DB
}

// NewGormStrategyRepository 创建策略仓储
func NewGormStrategyRepository(db *gorm.DB) *GormStrategyRepository {
	return &GormStrategyRepository{db: db}
}

// Save 保存策略
func (r *GormStrategyRepository) Save(ctx context.Context, strategy *domain.Strategy) error {
	return r.db.WithContext(ctx).Save(strategy).Error
}

// GetByID 根据ID获取
func (r *GormStrategyRepository) GetByID(ctx context.Context, id string) (*domain.Strategy, error) {
	var strategy domain.Strategy
	if err := r.db.WithContext(ctx).Where("strategy_id = ?", id).First(&strategy).Error; err != nil {
		return nil, fmt.Errorf("strategy not found: %w", err)
	}
	return &strategy, nil
}

// ListRunning 获取运行中的策略
func (r *GormStrategyRepository) ListRunning(ctx context.Context) ([]*domain.Strategy, error) {
	var strategies []*domain.Strategy
	if err := r.db.WithContext(ctx).Where("status = ?", domain.StrategyStatusRunning).Find(&strategies).Error; err != nil {
		return nil, err
	}
	return strategies, nil
}

// GormBacktestRepository GORM 回测仓储
type GormBacktestRepository struct {
	db *gorm.DB
}

// NewGormBacktestRepository 创建回测仓储
func NewGormBacktestRepository(db *gorm.DB) *GormBacktestRepository {
	return &GormBacktestRepository{db: db}
}

// Save 保存回测
func (r *GormBacktestRepository) Save(ctx context.Context, backtest *domain.Backtest) error {
	return r.db.WithContext(ctx).Save(backtest).Error
}

// GetByID 根据ID获取
func (r *GormBacktestRepository) GetByID(ctx context.Context, id string) (*domain.Backtest, error) {
	var backtest domain.Backtest
	if err := r.db.WithContext(ctx).Where("backtest_id = ?", id).First(&backtest).Error; err != nil {
		return nil, fmt.Errorf("backtest not found: %w", err)
	}
	return &backtest, nil
}
