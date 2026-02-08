// Package application 算法交易查询服务
package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/algotrading/domain"
)

// QueryService 算法交易查询服务
type QueryService struct {
	strategyRepo domain.StrategyRepository
	backtestRepo domain.BacktestRepository
	logger       *slog.Logger
}

// NewQueryService 创建查询服务
func NewQueryService(
	strategyRepo domain.StrategyRepository,
	backtestRepo domain.BacktestRepository,
	logger *slog.Logger,
) *QueryService {
	return &QueryService{
		strategyRepo: strategyRepo,
		backtestRepo: backtestRepo,
		logger:       logger,
	}
}

// StrategyDTO 策略 DTO
type StrategyDTO struct {
	StrategyID       string
	UserID           uint64
	Type             domain.StrategyType
	Status           domain.StrategyStatus
	Symbol           string
	Parameters       string
	ExecutedAmount   int64
	ExecutedQuantity int64
	CreatedAt        time.Time
}

// BacktestDTO 回测 DTO
type BacktestDTO struct {
	BacktestID string
	UserID     uint64
	Type       domain.StrategyType
	Symbol     string
	Parameters string
	StartTime  time.Time
	EndTime    time.Time
	Status     string
	ResultJSON string
}

// GetStrategy 获取策略
func (s *QueryService) GetStrategy(ctx context.Context, strategyID string) (*StrategyDTO, error) {
	strategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return nil, err
	}
	return s.toStrategyDTO(strategy), nil
}

// GetBacktestResult 获取回测结果
func (s *QueryService) GetBacktestResult(ctx context.Context, backtestID string) (*BacktestDTO, error) {
	backtest, err := s.backtestRepo.GetByID(ctx, backtestID)
	if err != nil {
		return nil, err
	}
	return s.toBacktestDTO(backtest), nil
}

func (s *QueryService) toStrategyDTO(strategy *domain.Strategy) *StrategyDTO {
	return &StrategyDTO{
		StrategyID:       strategy.StrategyID,
		UserID:           strategy.UserID,
		Type:             strategy.Type,
		Status:           strategy.Status,
		Symbol:           strategy.Symbol,
		Parameters:       strategy.Parameters,
		ExecutedAmount:   strategy.ExecutedAmount,
		ExecutedQuantity: strategy.ExecutedQuantity,
		CreatedAt:        strategy.CreatedAt,
	}
}

func (s *QueryService) toBacktestDTO(bt *domain.Backtest) *BacktestDTO {
	return &BacktestDTO{
		BacktestID: bt.BacktestID,
		UserID:     bt.UserID,
		Type:       bt.Type,
		Symbol:     bt.Symbol,
		Parameters: bt.Parameters,
		StartTime:  bt.StartTime,
		EndTime:    bt.EndTime,
		Status:     bt.Status,
		ResultJSON: bt.ResultJSON,
	}
}
