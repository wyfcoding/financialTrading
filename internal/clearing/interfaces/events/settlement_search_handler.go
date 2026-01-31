package events

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
)

type SettlementSearchHandler struct {
	repo       domain.SettlementRepository
	searchRepo domain.SettlementSearchRepository
}

func NewSettlementSearchHandler(repo domain.SettlementRepository, searchRepo domain.SettlementSearchRepository) *SettlementSearchHandler {
	return &SettlementSearchHandler{
		repo:       repo,
		searchRepo: searchRepo,
	}
}

func (h *SettlementSearchHandler) HandleSettlementCompleted(ctx context.Context, event domain.SettlementCompletedEvent) error {
	slog.InfoContext(ctx, "syncing settlement to es", "settlement_id", event.SettlementID)

	settlement, err := h.repo.Get(ctx, event.SettlementID)
	if err != nil {
		return err
	}
	if settlement == nil {
		return nil
	}

	return h.searchRepo.Index(ctx, settlement)
}
