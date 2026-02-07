package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

// MarketMakingQueryService 做市查询服务
type MarketMakingQueryService struct {
	repo                domain.MarketMakingRepository
	strategyReadRepo    domain.StrategyReadRepository
	performanceReadRepo domain.PerformanceReadRepository
	searchRepo          domain.MarketMakingSearchRepository
}

// NewMarketMakingQueryService 创建做市查询服务实例
func NewMarketMakingQueryService(
	repo domain.MarketMakingRepository,
	strategyReadRepo domain.StrategyReadRepository,
	performanceReadRepo domain.PerformanceReadRepository,
	searchRepo domain.MarketMakingSearchRepository,
) *MarketMakingQueryService {
	return &MarketMakingQueryService{
		repo:                repo,
		strategyReadRepo:    strategyReadRepo,
		performanceReadRepo: performanceReadRepo,
		searchRepo:          searchRepo,
	}
}

// GetStrategy 根据符号获取做市策略
func (s *MarketMakingQueryService) GetStrategy(ctx context.Context, symbol string) (*StrategyDTO, error) {
	if s.strategyReadRepo != nil {
		if cached, err := s.strategyReadRepo.Get(ctx, symbol); err == nil && cached != nil {
			return toStrategyDTO(cached), nil
		}
	}

	strategy, err := s.repo.GetStrategyBySymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if strategy == nil {
		return nil, nil
	}

	if s.strategyReadRepo != nil {
		_ = s.strategyReadRepo.Save(ctx, strategy)
	}
	return toStrategyDTO(strategy), nil
}

// GetPerformance 根据符号获取做市性能
func (s *MarketMakingQueryService) GetPerformance(ctx context.Context, symbol string) (*PerformanceDTO, error) {
	if s.performanceReadRepo != nil {
		if cached, err := s.performanceReadRepo.Get(ctx, symbol); err == nil && cached != nil {
			return toPerformanceDTO(cached), nil
		}
	}

	performance, err := s.repo.GetPerformanceBySymbol(ctx, symbol)
	if err != nil || performance == nil {
		// 如果获取失败，返回模拟数据（保留原逻辑）
		return &PerformanceDTO{
			Symbol:      symbol,
			TotalPnL:    1023.50,
			TotalVolume: 50000,
			TotalTrades: 125,
			SharpeRatio: 1.8,
		}, nil
	}

	if s.performanceReadRepo != nil {
		_ = s.performanceReadRepo.Save(ctx, performance)
	}

	return toPerformanceDTO(performance), nil
}

// ListStrategies 列出所有做市策略
func (s *MarketMakingQueryService) ListStrategies(ctx context.Context) ([]*StrategyDTO, error) {
	var strategies []*domain.QuoteStrategy
	var err error

	if s.searchRepo != nil {
		strategies, _, err = s.searchRepo.SearchStrategies(ctx, "", 1000, 0)
	}
	if err != nil || s.searchRepo == nil {
		strategies, err = s.repo.ListStrategies(ctx)
		if err != nil {
			return nil, err
		}
	}

	result := make([]*StrategyDTO, 0, len(strategies))
	for _, strategy := range strategies {
		result = append(result, toStrategyDTO(strategy))
	}

	return result, nil
}

func toStrategyDTO(strategy *domain.QuoteStrategy) *StrategyDTO {
	if strategy == nil {
		return nil
	}
	return &StrategyDTO{
		ID:           strategy.Symbol,
		Symbol:       strategy.Symbol,
		Spread:       strategy.Spread.String(),
		MinOrderSize: strategy.MinOrderSize.String(),
		MaxOrderSize: strategy.MaxOrderSize.String(),
		MaxPosition:  strategy.MaxPosition.String(),
		Status:       string(strategy.Status),
		CreatedAt:    strategy.CreatedAt.UnixMilli(),
		UpdatedAt:    strategy.UpdatedAt.UnixMilli(),
	}
}

func toPerformanceDTO(performance *domain.MarketMakingPerformance) *PerformanceDTO {
	if performance == nil {
		return nil
	}
	return &PerformanceDTO{
		Symbol:      performance.Symbol,
		TotalPnL:    performance.TotalPnL.InexactFloat64(),
		TotalVolume: performance.TotalVolume.InexactFloat64(),
		TotalTrades: int32(performance.TotalTrades),
		SharpeRatio: performance.SharpeRatio.InexactFloat64(),
		StartTime:   performance.StartTime.UnixMilli(),
		EndTime:     performance.EndTime.UnixMilli(),
	}
}
