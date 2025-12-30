// Package mysql 提供了量化策略和回测结果仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"gorm.io/gorm"
)

// StrategyModel 策略数据库模型
type StrategyModel struct {
	gorm.Model
	ID          string `gorm:"column:id;type:varchar(32);primaryKey"`
	Name        string `gorm:"column:name;type:varchar(100);not null"`
	Description string `gorm:"column:description;type:text"`
	Script      string `gorm:"column:script;type:text"`
	Status      string `gorm:"column:status;type:varchar(20);default:'ACTIVE'"`
}

func (StrategyModel) TableName() string { return "strategies" }

// BacktestResultModel 回测结果数据库模型
type BacktestResultModel struct {
	gorm.Model
	ID          string `gorm:"column:id;type:varchar(32);primaryKey"`
	StrategyID  string `gorm:"column:strategy_id;type:varchar(32);index;not null"`
	Symbol      string `gorm:"column:symbol;type:varchar(32);not null"`
	StartTime   int64  `gorm:"column:start_time;type:bigint"`
	EndTime     int64  `gorm:"column:end_time;type:bigint"`
	TotalReturn string `gorm:"column:total_return;type:decimal(32,18)"`
	MaxDrawdown string `gorm:"column:max_drawdown;type:decimal(32,18)"`
	SharpeRatio string `gorm:"column:sharpe_ratio;type:decimal(32,18)"`
	TotalTrades int    `gorm:"column:total_trades;type:int"`
	Status      string `gorm:"column:status;type:varchar(20);default:'RUNNING'"`
}

func (BacktestResultModel) TableName() string { return "backtest_results" }

type strategyRepositoryImpl struct {
	db *gorm.DB
}

func NewStrategyRepository(db *gorm.DB) domain.StrategyRepository {
	return &strategyRepositoryImpl{db: db}
}

func (r *strategyRepositoryImpl) Save(ctx context.Context, s *domain.Strategy) error {
	m := &StrategyModel{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Script:      s.Script,
		Status:      string(s.Status),
	}
	m.Model = s.Model
	err := r.db.WithContext(ctx).Save(m).Error
	if err == nil {
		s.Model = m.Model
	}
	return err
}

func (r *strategyRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.Strategy, error) {
	var m StrategyModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.Strategy{
		Model:       m.Model,
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Script:      m.Script,
		Status:      domain.StrategyStatus(m.Status),
	}, nil
}

type backtestResultRepositoryImpl struct {
	db *gorm.DB
}

func NewBacktestResultRepository(db *gorm.DB) domain.BacktestResultRepository {
	return &backtestResultRepositoryImpl{db: db}
}

func (r *backtestResultRepositoryImpl) Save(ctx context.Context, res *domain.BacktestResult) error {
	m := &BacktestResultModel{
		ID:          res.ID,
		StrategyID:  res.StrategyID,
		Symbol:      res.Symbol,
		StartTime:   res.StartTime,
		EndTime:     res.EndTime,
		TotalReturn: res.TotalReturn.String(),
		MaxDrawdown: res.MaxDrawdown.String(),
		SharpeRatio: res.SharpeRatio.String(),
		TotalTrades: res.TotalTrades,
		Status:      string(res.Status),
	}
	m.Model = res.Model
	err := r.db.WithContext(ctx).Save(m).Error
	if err == nil {
		res.Model = m.Model
	}
	return err
}

func (r *backtestResultRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.BacktestResult, error) {
	var m BacktestResultModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	totalReturn, err := decimal.NewFromString(m.TotalReturn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse total return: %w", err)
	}
	maxDrawdown, err := decimal.NewFromString(m.MaxDrawdown)
	if err != nil {
		return nil, fmt.Errorf("failed to parse max drawdown: %w", err)
	}
	sharpeRatio, err := decimal.NewFromString(m.SharpeRatio)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sharpe ratio: %w", err)
	}

	return &domain.BacktestResult{
		Model:       m.Model,
		ID:          m.ID,
		StrategyID:  m.StrategyID,
		Symbol:      m.Symbol,
		StartTime:   m.StartTime,
		EndTime:     m.EndTime,
		TotalReturn: totalReturn,
		MaxDrawdown: maxDrawdown,
		SharpeRatio: sharpeRatio,
		TotalTrades: m.TotalTrades,
		Status:      domain.BacktestStatus(m.Status),
	}, nil
}
