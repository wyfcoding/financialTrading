package application

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/quant/domain"
)

// QuantService 量化门面服务，整合 Manager 和 Query。
type QuantService struct {
	manager *QuantManager
	query   *QuantQuery
}

// NewQuantService 构造函数。
func NewQuantService(strategyRepo domain.StrategyRepository, backtestRepo domain.BacktestResultRepository, signalRepo domain.SignalRepository, marketDataClient domain.MarketDataClient) *QuantService {
	return &QuantService{
		manager: NewQuantManager(strategyRepo, backtestRepo, signalRepo, marketDataClient),
		query:   NewQuantQuery(strategyRepo, backtestRepo, signalRepo),
	}
}

// --- Manager (Writes) ---

func (s *QuantService) CreateStrategy(ctx context.Context, name string, description string, script string) (string, error) {
	return s.manager.CreateStrategy(ctx, name, description, script)
}

func (s *QuantService) RunBacktest(ctx context.Context, strategyID string, symbol string, startTime, endTime time.Time, initialCapital float64) (string, error) {
	return s.manager.RunBacktest(ctx, strategyID, symbol, startTime, endTime, initialCapital)
}

// --- Query (Reads) ---

func (s *QuantService) GetStrategy(ctx context.Context, id string) (*domain.Strategy, error) {
	return s.query.GetStrategy(ctx, id)
}

func (s *QuantService) GetBacktestResult(ctx context.Context, id string) (*domain.BacktestResult, error) {
	return s.query.GetBacktestResult(ctx, id)
}

func (s *QuantService) GetSignal(ctx context.Context, symbol string, indicator string, period int) (*SignalDTO, error) {
	return s.query.GetSignal(ctx, symbol, indicator, period)
}
