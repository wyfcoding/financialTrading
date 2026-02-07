package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/wyfcoding/financialtrading/internal/clearing/application"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
)

type SettlementProjectionHandler struct {
	projector *application.ClearingProjectionService
	logger    *slog.Logger
}

func NewSettlementProjectionHandler(projector *application.ClearingProjectionService, logger *slog.Logger) *SettlementProjectionHandler {
	return &SettlementProjectionHandler{projector: projector, logger: logger}
}

func (h *SettlementProjectionHandler) Handle(ctx context.Context, msg kafka.Message) error {
	switch msg.Topic {
	case domain.SettlementCreatedEventType:
		var payload struct {
			SettlementID string `json:"settlement_id"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal settlement created event", "error", err)
			return err
		}
		if payload.SettlementID == "" {
			return nil
		}
		return h.projector.Refresh(ctx, payload.SettlementID, false)
	case domain.SettlementCompletedEventType, domain.SettlementFailedEventType:
		var payload struct {
			SettlementID string `json:"settlement_id"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal settlement event", "error", err)
			return err
		}
		if payload.SettlementID == "" {
			return nil
		}
		return h.projector.Refresh(ctx, payload.SettlementID, true)
	default:
		h.logger.WarnContext(ctx, "unknown clearing event topic", "topic", msg.Topic)
		return nil
	}
}
