package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type marketMakingRepository struct {
	db *gorm.DB
}

// NewMarketMakingRepository 创建做市服务仓储实例
func NewMarketMakingRepository(db *gorm.DB) domain.MarketMakingRepository {
	return &marketMakingRepository{db: db}
}

// --- tx helpers ---

func (r *marketMakingRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *marketMakingRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *marketMakingRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *marketMakingRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

// --- Strategy ---

func (r *marketMakingRepository) SaveStrategy(ctx context.Context, strategy *domain.QuoteStrategy) error {
	model := toStrategyModel(strategy)
	if model == nil {
		return nil
	}

	db := r.getDB(ctx).WithContext(ctx)
	if model.ID == 0 {
		var existing StrategyModel
		err := db.Where("symbol = ?", model.Symbol).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(model).Error; err != nil {
				return err
			}
			strategy.ID = model.ID
			strategy.CreatedAt = model.CreatedAt
			strategy.UpdatedAt = model.UpdatedAt
			return nil
		}
		if err != nil {
			return err
		}
		model.ID = existing.ID
	}

	return db.Model(&StrategyModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"symbol":         model.Symbol,
			"spread":         model.Spread,
			"min_order_size": model.MinOrderSize,
			"max_order_size": model.MaxOrderSize,
			"max_position":   model.MaxPosition,
			"status":         model.Status,
		}).Error
}

func (r *marketMakingRepository) GetStrategyBySymbol(ctx context.Context, symbol string) (*domain.QuoteStrategy, error) {
	var model StrategyModel
	err := r.getDB(ctx).WithContext(ctx).Where("symbol = ?", symbol).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toStrategy(&model), err
}

func (r *marketMakingRepository) ListStrategies(ctx context.Context) ([]*domain.QuoteStrategy, error) {
	var models []*StrategyModel
	err := r.getDB(ctx).WithContext(ctx).Find(&models).Error
	if err != nil {
		return nil, err
	}
	strategies := make([]*domain.QuoteStrategy, len(models))
	for i, m := range models {
		strategies[i] = toStrategy(m)
	}
	return strategies, nil
}

// --- Performance ---

func (r *marketMakingRepository) SavePerformance(ctx context.Context, p *domain.MarketMakingPerformance) error {
	model := toPerformanceModel(p)
	if model == nil {
		return nil
	}

	db := r.getDB(ctx).WithContext(ctx)
	if model.ID == 0 {
		var existing PerformanceModel
		err := db.Where("symbol = ?", model.Symbol).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(model).Error; err != nil {
				return err
			}
			p.ID = model.ID
			p.CreatedAt = model.CreatedAt
			p.UpdatedAt = model.UpdatedAt
			return nil
		}
		if err != nil {
			return err
		}
		model.ID = existing.ID
	}

	return db.Model(&PerformanceModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"symbol":       model.Symbol,
			"total_pnl":    model.TotalPnL,
			"total_volume": model.TotalVolume,
			"total_trades": model.TotalTrades,
			"sharpe_ratio": model.SharpeRatio,
			"start_time":   model.StartTime,
			"end_time":     model.EndTime,
		}).Error
}

func (r *marketMakingRepository) GetPerformanceBySymbol(ctx context.Context, symbol string) (*domain.MarketMakingPerformance, error) {
	var model PerformanceModel
	err := r.getDB(ctx).WithContext(ctx).Where("symbol = ?", symbol).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toPerformance(&model), err
}

func (r *marketMakingRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
