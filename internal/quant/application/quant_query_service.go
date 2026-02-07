package application

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/quant/arbitrage"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
)

// QuantQueryService 处理所有量化相关的查询操作（读模型）
type QuantQueryService struct {
	strategyRepo     domain.StrategyRepository
	strategyReadRepo domain.StrategyReadRepository
	backtestRepo     domain.BacktestResultRepository
	backtestReadRepo domain.BacktestResultReadRepository
	signalRepo       domain.SignalRepository
	signalReadRepo   domain.SignalReadRepository
	searchRepo       domain.QuantSearchRepository
	arbitrageEngine  *arbitrage.Engine
}

// NewQuantQueryService 构造函数。
func NewQuantQueryService(
	strategyRepo domain.StrategyRepository,
	strategyReadRepo domain.StrategyReadRepository,
	backtestRepo domain.BacktestResultRepository,
	backtestReadRepo domain.BacktestResultReadRepository,
	signalRepo domain.SignalRepository,
	signalReadRepo domain.SignalReadRepository,
	searchRepo domain.QuantSearchRepository,
	arbitrageEngine *arbitrage.Engine,
) *QuantQueryService {
	return &QuantQueryService{
		strategyRepo:     strategyRepo,
		strategyReadRepo: strategyReadRepo,
		backtestRepo:     backtestRepo,
		backtestReadRepo: backtestReadRepo,
		signalRepo:       signalRepo,
		signalReadRepo:   signalReadRepo,
		searchRepo:       searchRepo,
		arbitrageEngine:  arbitrageEngine,
	}
}

// GetStrategy 获取策略
func (q *QuantQueryService) GetStrategy(ctx context.Context, id string) (*domain.Strategy, error) {
	if q.strategyReadRepo != nil {
		if cached, err := q.strategyReadRepo.Get(ctx, id); err == nil && cached != nil {
			return cached, nil
		}
	}
	strategy, err := q.strategyRepo.GetByID(ctx, id)
	if err != nil || strategy == nil {
		return nil, err
	}
	if q.strategyReadRepo != nil {
		_ = q.strategyReadRepo.Save(ctx, strategy)
	}
	return strategy, nil
}

// GetBacktestResult 获取回测结果
func (q *QuantQueryService) GetBacktestResult(ctx context.Context, id string) (*domain.BacktestResult, error) {
	if q.backtestReadRepo != nil {
		if cached, err := q.backtestReadRepo.Get(ctx, id); err == nil && cached != nil {
			return cached, nil
		}
	}
	result, err := q.backtestRepo.GetByID(ctx, id)
	if err != nil || result == nil {
		return nil, err
	}
	if q.backtestReadRepo != nil {
		_ = q.backtestReadRepo.Save(ctx, result)
	}
	return result, nil
}

// GetSignal 获取信号
func (q *QuantQueryService) GetSignal(ctx context.Context, symbol string, indicator string, period int) (*SignalDTO, error) {
	indType := domain.IndicatorType(indicator)
	if q.signalReadRepo != nil {
		if cached, err := q.signalReadRepo.GetLatest(ctx, symbol, indType, period); err == nil && cached != nil {
			return toSignalDTO(cached), nil
		}
	}

	signal, err := q.signalRepo.GetLatest(ctx, symbol, indType, period)
	if err == nil && signal != nil {
		if q.signalReadRepo != nil {
			_ = q.signalReadRepo.Save(ctx, signal)
		}
		return toSignalDTO(signal), nil
	}

	// fallback mock signal
	mock := &domain.Signal{
		Symbol:    symbol,
		Indicator: indType,
		Period:    period,
		Value:     100.0,
		Timestamp: time.Now(),
	}
	return toSignalDTO(mock), nil
}

// SearchStrategies 策略搜索 (Elasticsearch)
func (q *QuantQueryService) SearchStrategies(ctx context.Context, keyword string, status string, limit, offset int) ([]*domain.Strategy, int64, error) {
	if q.searchRepo == nil {
		return nil, 0, nil
	}
	var st domain.StrategyStatus
	if status != "" {
		st = domain.StrategyStatus(status)
	}
	return q.searchRepo.SearchStrategies(ctx, keyword, st, limit, offset)
}

// SearchBacktestResults 回测搜索 (Elasticsearch)
func (q *QuantQueryService) SearchBacktestResults(ctx context.Context, symbol string, status string, limit, offset int) ([]*domain.BacktestResult, int64, error) {
	if q.searchRepo == nil {
		return nil, 0, nil
	}
	var st domain.BacktestStatus
	if status != "" {
		st = domain.BacktestStatus(status)
	}
	return q.searchRepo.SearchBacktestResults(ctx, symbol, st, limit, offset)
}

// FindArbitrageOpportunities 套利机会查询
func (q *QuantQueryService) FindArbitrageOpportunities(ctx context.Context, symbol string, venues []string) ([]*ArbitrageOpportunityDTO, error) {
	if q.arbitrageEngine == nil {
		return nil, nil
	}
	if symbol == "" || len(venues) == 0 {
		return nil, nil
	}
	opps, err := q.arbitrageEngine.FindOpportunities(ctx, symbol, venues)
	if err != nil {
		return nil, err
	}
	dtos := make([]*ArbitrageOpportunityDTO, 0, len(opps))
	for _, opp := range opps {
		dtos = append(dtos, &ArbitrageOpportunityDTO{
			Symbol:      opp.Symbol,
			BuyVenue:    opp.BuyVenue,
			SellVenue:   opp.SellVenue,
			Spread:      opp.Spread.String(),
			MaxQuantity: opp.MaxQuantity,
		})
	}
	return dtos, nil
}
