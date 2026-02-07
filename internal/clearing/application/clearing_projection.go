package application

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
)

// ClearingProjectionService 负责将结算写模型投影到读模型（Redis/ES）。
type ClearingProjectionService struct {
	repo       domain.SettlementRepository
	readRepo   domain.SettlementReadRepository
	searchRepo domain.SettlementSearchRepository
	logger     *slog.Logger
}

func NewClearingProjectionService(
	repo domain.SettlementRepository,
	readRepo domain.SettlementReadRepository,
	searchRepo domain.SettlementSearchRepository,
	logger *slog.Logger,
) *ClearingProjectionService {
	return &ClearingProjectionService{
		repo:       repo,
		readRepo:   readRepo,
		searchRepo: searchRepo,
		logger:     logger,
	}
}

// Refresh 重新投影指定结算单到读模型。
func (s *ClearingProjectionService) Refresh(ctx context.Context, settlementID string, syncSearch bool) error {
	if settlementID == "" {
		return nil
	}
	settlement, err := s.repo.Get(ctx, settlementID)
	if err != nil {
		return err
	}
	if settlement == nil {
		return nil
	}
	if s.readRepo != nil {
		if err := s.readRepo.Save(ctx, settlement); err != nil {
			s.logger.WarnContext(ctx, "failed to update settlement redis cache", "error", err, "settlement_id", settlementID)
		}
	}
	if syncSearch && s.searchRepo != nil {
		if err := s.searchRepo.Index(ctx, settlement); err != nil {
			s.logger.WarnContext(ctx, "failed to index settlement to search", "error", err, "settlement_id", settlementID)
			return err
		}
	}
	return nil
}
