package application

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
)

// MatchingProjectionService 负责将撮合结果投影到读模型（Redis/ES）。
type MatchingProjectionService struct {
	tradeReadRepo   domain.TradeReadRepository
	tradeSearchRepo domain.TradeSearchRepository
	logger          *slog.Logger
}

func NewMatchingProjectionService(
	tradeReadRepo domain.TradeReadRepository,
	tradeSearchRepo domain.TradeSearchRepository,
	logger *slog.Logger,
) *MatchingProjectionService {
	return &MatchingProjectionService{
		tradeReadRepo:   tradeReadRepo,
		tradeSearchRepo: tradeSearchRepo,
		logger:          logger,
	}
}

func (s *MatchingProjectionService) ProjectTrade(ctx context.Context, trade *domain.Trade) error {
	if trade == nil {
		return nil
	}
	if s.tradeReadRepo != nil {
		if err := s.tradeReadRepo.Save(ctx, trade); err != nil {
			s.logger.WarnContext(ctx, "failed to update trade cache", "error", err, "trade_id", trade.TradeID)
			return err
		}
	}
	if s.tradeSearchRepo != nil {
		if err := s.tradeSearchRepo.Index(ctx, trade); err != nil {
			s.logger.WarnContext(ctx, "failed to index trade", "error", err, "trade_id", trade.TradeID)
			return err
		}
	}
	return nil
}
