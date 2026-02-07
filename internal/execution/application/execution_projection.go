package application

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

// ExecutionProjectionService 将交易与算法读模型投影到 Redis/ES。
type ExecutionProjectionService struct {
	tradeRepo     domain.TradeRepository
	tradeReadRepo domain.TradeReadRepository
	searchRepo    domain.TradeSearchRepository
	algoRepo      domain.AlgoOrderRepository
	algoReadRepo  domain.AlgoRedisRepository
	logger        *slog.Logger
}

func NewExecutionProjectionService(
	tradeRepo domain.TradeRepository,
	tradeReadRepo domain.TradeReadRepository,
	searchRepo domain.TradeSearchRepository,
	algoRepo domain.AlgoOrderRepository,
	algoReadRepo domain.AlgoRedisRepository,
	logger *slog.Logger,
) *ExecutionProjectionService {
	return &ExecutionProjectionService{
		tradeRepo:     tradeRepo,
		tradeReadRepo: tradeReadRepo,
		searchRepo:    searchRepo,
		algoRepo:      algoRepo,
		algoReadRepo:  algoReadRepo,
		logger:        logger,
	}
}

// RefreshTrade 刷新成交读模型并同步搜索索引。
func (s *ExecutionProjectionService) RefreshTrade(ctx context.Context, tradeID string, syncSearch bool) error {
	if tradeID == "" {
		return nil
	}
	trade, err := s.tradeRepo.Get(ctx, tradeID)
	if err != nil {
		return err
	}
	if trade == nil {
		return nil
	}
	if s.tradeReadRepo != nil {
		if err := s.tradeReadRepo.Save(ctx, trade); err != nil {
			s.logger.WarnContext(ctx, "failed to cache trade", "trade_id", tradeID, "error", err)
		}
	}
	if syncSearch && s.searchRepo != nil {
		if err := s.searchRepo.Index(ctx, trade); err != nil {
			s.logger.WarnContext(ctx, "failed to index trade", "trade_id", tradeID, "error", err)
			return err
		}
	}
	return nil
}

// RefreshAlgo 刷新算法订单读模型（Redis）。
func (s *ExecutionProjectionService) RefreshAlgo(ctx context.Context, algoID string) error {
	if algoID == "" {
		return nil
	}
	order, err := s.algoRepo.Get(ctx, algoID)
	if err != nil {
		return err
	}
	if order == nil {
		return nil
	}
	if s.algoReadRepo != nil {
		if err := s.algoReadRepo.Save(ctx, order); err != nil {
			s.logger.WarnContext(ctx, "failed to cache algo order", "algo_id", algoID, "error", err)
		}
	}
	return nil
}
