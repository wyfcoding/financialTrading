// 生成摘要：实现算法交易服务的 MySQL 仓储层，基于 GORM。
// 变更说明：从旧的 infrastructure 目录迁移至 persistence/mysql，明确技术实现边界。

package mysql

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/algotrading/domain"
	"gorm.io/gorm"
)

// strategyRepository GORM 策略仓储实现
type strategyRepository struct {
	db *gorm.DB
}

// NewStrategyRepository 创建策略仓储
func NewStrategyRepository(db *gorm.DB) domain.StrategyRepository {
	return &strategyRepository{db: db}
}

// Save 保存策略聚合根
func (r *strategyRepository) Save(ctx context.Context, strategy *domain.Strategy) error {
	return r.db.WithContext(ctx).Save(strategy).Error
}

// GetByID 根据业务 ID 获取策略
func (r *strategyRepository) GetByID(ctx context.Context, id string) (*domain.Strategy, error) {
	var strategy domain.Strategy
	if err := r.db.WithContext(ctx).Where("strategy_id = ?", id).First(&strategy).Error; err != nil {
		return nil, fmt.Errorf("strategy not found: %w", err)
	}
	return &strategy, nil
}

// ListRunning 获取所有运行中的策略
func (r *strategyRepository) ListRunning(ctx context.Context) ([]*domain.Strategy, error) {
	var strategies []*domain.Strategy
	if err := r.db.WithContext(ctx).Where("status = ?", domain.StrategyStatusRunning).Find(&strategies).Error; err != nil {
		return nil, err
	}
	return strategies, nil
}

// backtestRepository GORM 回测仓储实现
type backtestRepository struct {
	db *gorm.DB
}

// NewBacktestRepository 创建回测仓储
func NewBacktestRepository(db *gorm.DB) domain.BacktestRepository {
	return &backtestRepository{db: db}
}

// Save 保存回测任务实体
func (r *backtestRepository) Save(ctx context.Context, backtest *domain.Backtest) error {
	return r.db.WithContext(ctx).Save(backtest).Error
}

// GetByID 根据业务 ID 获取回测任务
func (r *backtestRepository) GetByID(ctx context.Context, id string) (*domain.Backtest, error) {
	var backtest domain.Backtest
	if err := r.db.WithContext(ctx).Where("backtest_id = ?", id).First(&backtest).Error; err != nil {
		return nil, fmt.Errorf("backtest not found: %w", err)
	}
	return &backtest, nil
}
