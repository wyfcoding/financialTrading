package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/wyfcoding/financialTrading/internal/quant/domain"
	"github.com/wyfcoding/financialTrading/pkg/db"
	"gorm.io/gorm"
)

// StrategyModel 策略数据库模型
type StrategyModel struct {
	gorm.Model
	ID          string `gorm:"column:id;type:varchar(36);primaryKey"`
	Name        string `gorm:"column:name;type:varchar(100);not null"`
	Description string `gorm:"column:description;type:text"`
	Script      string `gorm:"column:script;type:text"`
	Status      string `gorm:"column:status;type:varchar(20);default:'ACTIVE'"`
}

// TableName 指定表名
func (StrategyModel) TableName() string {
	return "strategies"
}

// StrategyRepositoryImpl 策略仓储实现
type StrategyRepositoryImpl struct {
	db *db.DB
}

// NewStrategyRepository 创建策略仓储实例
func NewStrategyRepository(db *db.DB) domain.StrategyRepository {
	return &StrategyRepositoryImpl{db: db}
}

// Save 保存策略
func (r *StrategyRepositoryImpl) Save(ctx context.Context, strategy *domain.Strategy) error {
	model := &StrategyModel{
		Model:       strategy.Model,
		ID:          strategy.ID,
		Name:        strategy.Name,
		Description: strategy.Description,
		Script:      strategy.Script,
		Status:      string(strategy.Status),
	}

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return fmt.Errorf("failed to save strategy: %w", err)
	}

	strategy.Model = model.Model
	return nil
}

// GetByID 根据 ID 获取策略
func (r *StrategyRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.Strategy, error) {
	var model StrategyModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get strategy: %w", err)
	}

	return &domain.Strategy{
		Model:       model.Model,
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		Script:      model.Script,
		Status:      domain.StrategyStatus(model.Status),
	}, nil
}

// BacktestResultModel 回测结果数据库模型
type BacktestResultModel struct {
	gorm.Model
	ID          string    `gorm:"column:id;type:varchar(36);primaryKey"`
	StrategyID  string    `gorm:"column:strategy_id;type:varchar(36);index;not null"`
	Symbol      string    `gorm:"column:symbol;type:varchar(20);not null"`
	StartTime   time.Time `gorm:"column:start_time;type:datetime"`
	EndTime     time.Time `gorm:"column:end_time;type:datetime"`
	TotalReturn float64   `gorm:"column:total_return;type:decimal(20,8)"`
	MaxDrawdown float64   `gorm:"column:max_drawdown;type:decimal(20,8)"`
	SharpeRatio float64   `gorm:"column:sharpe_ratio;type:decimal(20,8)"`
	TotalTrades int       `gorm:"column:total_trades;type:int"`
	Status      string    `gorm:"column:status;type:varchar(20);default:'RUNNING'"`
}

// TableName 指定表名
func (BacktestResultModel) TableName() string {
	return "backtest_results"
}

// BacktestResultRepositoryImpl 回测结果仓储实现
type BacktestResultRepositoryImpl struct {
	db *db.DB
}

// NewBacktestResultRepository 创建回测结果仓储实例
func NewBacktestResultRepository(db *db.DB) domain.BacktestResultRepository {
	return &BacktestResultRepositoryImpl{db: db}
}

// Save 保存回测结果
func (r *BacktestResultRepositoryImpl) Save(ctx context.Context, result *domain.BacktestResult) error {
	model := &BacktestResultModel{
		Model:       result.Model,
		ID:          result.ID,
		StrategyID:  result.StrategyID,
		Symbol:      result.Symbol,
		StartTime:   result.StartTime,
		EndTime:     result.EndTime,
		TotalReturn: result.TotalReturn,
		MaxDrawdown: result.MaxDrawdown,
		SharpeRatio: result.SharpeRatio,
		TotalTrades: result.TotalTrades,
		Status:      string(result.Status),
	}

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return fmt.Errorf("failed to save backtest result: %w", err)
	}

	result.Model = model.Model
	return nil
}

// GetByID 根据 ID 获取回测结果
func (r *BacktestResultRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.BacktestResult, error) {
	var model BacktestResultModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get backtest result: %w", err)
	}

	return &domain.BacktestResult{
		Model:       model.Model,
		ID:          model.ID,
		StrategyID:  model.StrategyID,
		Symbol:      model.Symbol,
		StartTime:   model.StartTime,
		EndTime:     model.EndTime,
		TotalReturn: model.TotalReturn,
		MaxDrawdown: model.MaxDrawdown,
		SharpeRatio: model.SharpeRatio,
		TotalTrades: model.TotalTrades,
		Status:      domain.BacktestStatus(model.Status),
	}, nil
}
