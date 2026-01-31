package application

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/quant/domain"
)

// QuantQuery 处理所有量化相关的查询操作（Queries）。
type QuantQuery struct {
	strategyRepo domain.StrategyRepository
	backtestRepo domain.BacktestResultRepository
	signalRepo   domain.SignalRepository
}

// NewQuantQuery 构造函数。
func NewQuantQuery(strategyRepo domain.StrategyRepository, backtestRepo domain.BacktestResultRepository, signalRepo domain.SignalRepository) *QuantQuery {
	return &QuantQuery{
		strategyRepo: strategyRepo,
		backtestRepo: backtestRepo,
		signalRepo:   signalRepo,
	}
}

// GetStrategy 获取策略
func (q *QuantQuery) GetStrategy(ctx context.Context, id string) (*domain.Strategy, error) {
	return q.strategyRepo.GetByID(ctx, id)
}

// GetBacktestResult 获取回测结果
func (q *QuantQuery) GetBacktestResult(ctx context.Context, id string) (*domain.BacktestResult, error) {
	return q.backtestRepo.GetByID(ctx, id)
}

// GetSignal 获取信号
func (q *QuantQuery) GetSignal(ctx context.Context, symbol string, indicator string, period int) (*SignalDTO, error) {
	indType := domain.IndicatorType(indicator)
	signal, err := q.signalRepo.GetLatest(ctx, symbol, indType, period)
	if err != nil {
		// Mock calculation for simplicity if not found
		signal = &domain.Signal{
			Symbol:    symbol,
			Indicator: indType,
			Period:    period,
			Value:     100.0, // Default
			Timestamp: time.Now(),
		}
	}
	return &SignalDTO{
		Symbol:    signal.Symbol,
		Indicator: string(signal.Indicator),
		Period:    signal.Period,
		Value:     signal.Value,
		Timestamp: signal.Timestamp.Unix(),
	}, nil
}
