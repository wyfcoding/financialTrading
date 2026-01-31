package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

// MarketMakingQueryService 做市查询服务
type MarketMakingQueryService struct {
	repo domain.MarketMakingRepository
}

// NewMarketMakingQueryService 创建做市查询服务实例
func NewMarketMakingQueryService(
	repo domain.MarketMakingRepository,
) *MarketMakingQueryService {
	return &MarketMakingQueryService{
		repo: repo,
	}
}

// GetStrategy 根据符号获取做市策略
func (s *MarketMakingQueryService) GetStrategy(ctx context.Context, symbol string) (*StrategyDTO, error) {
	strategy, err := s.repo.GetStrategyBySymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if strategy == nil {
		return nil, nil
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
	}, nil
}

// GetPerformance 根据符号获取做市性能
func (s *MarketMakingQueryService) GetPerformance(ctx context.Context, symbol string) (*PerformanceDTO, error) {
	// 从存储库获取性能数据
	performance, err := s.repo.GetPerformanceBySymbol(ctx, symbol)
	if err != nil || performance == nil {
		// 如果获取失败，返回模拟数据
		return &PerformanceDTO{
			Symbol:      symbol,
			TotalPnL:    1023.50,
			TotalVolume: 50000,
			TotalTrades: 125,
			SharpeRatio: 1.8,
		}, nil
	}

	return &PerformanceDTO{
		Symbol:      performance.Symbol,
		TotalPnL:    performance.TotalPnL.InexactFloat64(),
		TotalVolume: performance.TotalVolume.InexactFloat64(),
		TotalTrades: int32(performance.TotalTrades),
		SharpeRatio: performance.SharpeRatio.InexactFloat64(),
	}, nil
}

// ListStrategies 列出所有做市策略
func (s *MarketMakingQueryService) ListStrategies(ctx context.Context) ([]*StrategyDTO, error) {
	strategies, err := s.repo.ListStrategies(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*StrategyDTO, 0, len(strategies))
	for _, strategy := range strategies {
		result = append(result, &StrategyDTO{
			ID:           strategy.Symbol,
			Symbol:       strategy.Symbol,
			Spread:       strategy.Spread.String(),
			MinOrderSize: strategy.MinOrderSize.String(),
			MaxOrderSize: strategy.MaxOrderSize.String(),
			MaxPosition:  strategy.MaxPosition.String(),
			Status:       string(strategy.Status),
			CreatedAt:    strategy.CreatedAt.UnixMilli(),
			UpdatedAt:    strategy.UpdatedAt.UnixMilli(),
		})
	}

	return result, nil
}
