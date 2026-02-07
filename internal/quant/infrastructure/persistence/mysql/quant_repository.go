package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

// --- Strategy Repository ---

type strategyRepository struct {
	db *gorm.DB
}

func NewStrategyRepository(db *gorm.DB) domain.StrategyRepository {
	return &strategyRepository{db: db}
}

func (r *strategyRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *strategyRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *strategyRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *strategyRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *strategyRepository) Save(ctx context.Context, s *domain.Strategy) error {
	model := toStrategyModel(s)
	if model == nil {
		return nil
	}
	db := r.getDB(ctx).WithContext(ctx)
	if err := db.Save(model).Error; err != nil {
		return err
	}
	s.CreatedAt = model.CreatedAt
	s.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *strategyRepository) GetByID(ctx context.Context, id string) (*domain.Strategy, error) {
	var model StrategyModel
	err := r.getDB(ctx).WithContext(ctx).Where("id = ?", id).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toStrategy(&model), nil
}

func (r *strategyRepository) Delete(ctx context.Context, id string) error {
	return r.getDB(ctx).WithContext(ctx).Where("id = ?", id).Delete(&StrategyModel{}).Error
}

func (r *strategyRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := contextx.GetTx(ctx); tx != nil {
		if gormTx, ok := tx.(*gorm.DB); ok {
			return gormTx
		}
	}
	return r.db
}

// --- Backtest Repository ---

type backtestResultRepository struct {
	db *gorm.DB
}

func NewBacktestResultRepository(db *gorm.DB) domain.BacktestResultRepository {
	return &backtestResultRepository{db: db}
}

func (r *backtestResultRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *backtestResultRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *backtestResultRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *backtestResultRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *backtestResultRepository) Save(ctx context.Context, res *domain.BacktestResult) error {
	model := toBacktestResultModel(res)
	if model == nil {
		return nil
	}
	db := r.getDB(ctx).WithContext(ctx)
	if err := db.Save(model).Error; err != nil {
		return err
	}
	res.CreatedAt = model.CreatedAt
	res.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *backtestResultRepository) GetByID(ctx context.Context, id string) (*domain.BacktestResult, error) {
	var model BacktestResultModel
	err := r.getDB(ctx).WithContext(ctx).Where("id = ?", id).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result, err := toBacktestResult(&model)
	if err != nil {
		return nil, fmt.Errorf("failed to parse backtest result: %w", err)
	}
	return result, nil
}

func (r *backtestResultRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := contextx.GetTx(ctx); tx != nil {
		if gormTx, ok := tx.(*gorm.DB); ok {
			return gormTx
		}
	}
	return r.db
}

// --- Signal Repository ---

type signalRepository struct {
	db *gorm.DB
}

func NewSignalRepository(db *gorm.DB) domain.SignalRepository {
	return &signalRepository{db: db}
}

func (r *signalRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *signalRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *signalRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *signalRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *signalRepository) Save(ctx context.Context, signal *domain.Signal) error {
	model := toSignalModel(signal)
	if model == nil {
		return nil
	}
	db := r.getDB(ctx).WithContext(ctx)
	if err := db.Create(model).Error; err != nil {
		return err
	}
	signal.ID = model.ID
	signal.CreatedAt = model.CreatedAt
	signal.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *signalRepository) GetLatest(ctx context.Context, symbol string, indicator domain.IndicatorType, period int) (*domain.Signal, error) {
	var model SignalModel
	err := r.getDB(ctx).WithContext(ctx).
		Where("symbol = ? AND indicator = ? AND period = ?", symbol, string(indicator), period).
		Order("timestamp desc").
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toSignal(&model), nil
}

func (r *signalRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := contextx.GetTx(ctx); tx != nil {
		if gormTx, ok := tx.(*gorm.DB); ok {
			return gormTx
		}
	}
	return r.db
}
