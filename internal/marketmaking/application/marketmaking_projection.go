package application

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

// MarketMakingProjectionService 负责将写模型投影到读模型（Redis/ES）。
type MarketMakingProjectionService struct {
	repo                domain.MarketMakingRepository
	strategyReadRepo    domain.StrategyReadRepository
	performanceReadRepo domain.PerformanceReadRepository
	searchRepo          domain.MarketMakingSearchRepository
	logger              *slog.Logger
}

func NewMarketMakingProjectionService(
	repo domain.MarketMakingRepository,
	strategyReadRepo domain.StrategyReadRepository,
	performanceReadRepo domain.PerformanceReadRepository,
	searchRepo domain.MarketMakingSearchRepository,
	logger *slog.Logger,
) *MarketMakingProjectionService {
	return &MarketMakingProjectionService{
		repo:                repo,
		strategyReadRepo:    strategyReadRepo,
		performanceReadRepo: performanceReadRepo,
		searchRepo:          searchRepo,
		logger:              logger,
	}
}

func (s *MarketMakingProjectionService) RefreshStrategy(ctx context.Context, symbol string, syncSearch bool) error {
	if symbol == "" {
		return nil
	}
	strategy, err := s.repo.GetStrategyBySymbol(ctx, symbol)
	if err != nil || strategy == nil {
		return err
	}
	if s.strategyReadRepo != nil {
		if err := s.strategyReadRepo.Save(ctx, strategy); err != nil {
			s.logger.WarnContext(ctx, "failed to update strategy cache", "error", err, "symbol", symbol)
		}
	}
	if syncSearch && s.searchRepo != nil {
		if err := s.searchRepo.IndexStrategy(ctx, strategy); err != nil {
			s.logger.WarnContext(ctx, "failed to index strategy", "error", err, "symbol", symbol)
			return err
		}
	}
	return nil
}

func (s *MarketMakingProjectionService) RefreshPerformance(ctx context.Context, symbol string, syncSearch bool) error {
	if symbol == "" {
		return nil
	}
	performance, err := s.repo.GetPerformanceBySymbol(ctx, symbol)
	if err != nil || performance == nil {
		return err
	}
	if s.performanceReadRepo != nil {
		if err := s.performanceReadRepo.Save(ctx, performance); err != nil {
			s.logger.WarnContext(ctx, "failed to update performance cache", "error", err, "symbol", symbol)
		}
	}
	if syncSearch && s.searchRepo != nil {
		if err := s.searchRepo.IndexPerformance(ctx, performance); err != nil {
			s.logger.WarnContext(ctx, "failed to index performance", "error", err, "symbol", symbol)
			return err
		}
	}
	return nil
}
