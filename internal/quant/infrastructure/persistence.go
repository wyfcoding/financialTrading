package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/wyfcoding/financialTrading/internal/quant/domain"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"gorm.io/gorm"
)

// StrategyModel 策略数据库模型
type StrategyModel struct {
	gorm.Model
	ID          string `gorm:"column:id;type:varchar(36);primaryKey;comment:策略ID"`
	Name        string `gorm:"column:name;type:varchar(100);not null;comment:策略名称"`
	Description string `gorm:"column:description;type:text;comment:策略描述"`
	Script      string `gorm:"column:script;type:text;comment:策略脚本"`
	Status      string `gorm:"column:status;type:varchar(20);default:'ACTIVE';comment:状态"`
}

// TableName 指定表名
func (StrategyModel) TableName() string {
	return "strategies"
}

// ToDomain 转换为领域实体
func (m *StrategyModel) ToDomain() *domain.Strategy {
	return &domain.Strategy{
		Model:       m.Model,
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Script:      m.Script,
		Status:      domain.StrategyStatus(m.Status),
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// BacktestResultModel 回测结果数据库模型
type BacktestResultModel struct {
	gorm.Model
	ID          string  `gorm:"column:id;type:varchar(36);primaryKey;comment:回测ID"`
	StrategyID  string  `gorm:"column:strategy_id;type:varchar(36);index;not null;comment:策略ID"`
	Symbol      string  `gorm:"column:symbol;type:varchar(20);not null;comment:交易对"`
	TotalReturn float64 `gorm:"column:total_return;type:decimal(20,8);comment:总收益"`
	MaxDrawdown float64 `gorm:"column:max_drawdown;type:decimal(20,8);comment:最大回撤"`
	SharpeRatio float64 `gorm:"column:sharpe_ratio;type:decimal(20,8);comment:夏普比率"`
	TotalTrades int     `gorm:"column:total_trades;type:int;comment:总交易次数"`
	Status      string  `gorm:"column:status;type:varchar(20);default:'RUNNING';comment:状态"`
}

// TableName 指定表名
func (BacktestResultModel) TableName() string {
	return "backtest_results"
}

// ToDomain 转换为领域实体
func (m *BacktestResultModel) ToDomain() *domain.BacktestResult {
	return &domain.BacktestResult{
		Model:       m.Model,
		ID:          m.ID,
		StrategyID:  m.StrategyID,
		Symbol:      m.Symbol,
		TotalReturn: m.TotalReturn,
		MaxDrawdown: m.MaxDrawdown,
		SharpeRatio: m.SharpeRatio,
		TotalTrades: m.TotalTrades,
		Status:      domain.BacktestStatus(m.Status),
		CreatedAt:   m.CreatedAt,
	}
}

// StrategyRepositoryImpl 策略仓储实现
type StrategyRepositoryImpl struct {
	db *gorm.DB
}

func NewStrategyRepository(db *gorm.DB) domain.StrategyRepository {
	return &StrategyRepositoryImpl{db: db}
}

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
		logger.Error(ctx, "Failed to save strategy",
			"strategy_id", strategy.ID,
			"error", err,
		)
		return fmt.Errorf("failed to save strategy: %w", err)
	}

	strategy.Model = model.Model
	return nil
}

func (r *StrategyRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.Strategy, error) {
	var model StrategyModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.Error(ctx, "Failed to get strategy by ID",
			"strategy_id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get strategy by ID: %w", err)
	}
	return model.ToDomain(), nil
}

// BacktestResultRepositoryImpl 回测结果仓储实现
type BacktestResultRepositoryImpl struct {
	db *gorm.DB
}

func NewBacktestResultRepository(db *gorm.DB) domain.BacktestResultRepository {
	return &BacktestResultRepositoryImpl{db: db}
}

func (r *BacktestResultRepositoryImpl) Save(ctx context.Context, result *domain.BacktestResult) error {
	model := &BacktestResultModel{
		Model:       result.Model,
		ID:          result.ID,
		StrategyID:  result.StrategyID,
		Symbol:      result.Symbol,
		TotalReturn: result.TotalReturn,
		MaxDrawdown: result.MaxDrawdown,
		SharpeRatio: result.SharpeRatio,
		TotalTrades: result.TotalTrades,
		Status:      string(result.Status),
	}
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		logger.Error(ctx, "Failed to save backtest result",
			"result_id", result.ID,
			"error", err,
		)
		return fmt.Errorf("failed to save backtest result: %w", err)
	}

	result.Model = model.Model
	return nil
}

func (r *BacktestResultRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.BacktestResult, error) {
	var model BacktestResultModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.Error(ctx, "Failed to get backtest result by ID",
			"result_id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get backtest result by ID: %w", err)
	}
	return model.ToDomain(), nil
}
