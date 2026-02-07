package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/wyfcoding/financialtrading/internal/account/application"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
)

type AccountProjectionHandler struct {
	projector *application.AccountProjectionService
	logger    *slog.Logger
}

func NewAccountProjectionHandler(projector *application.AccountProjectionService, logger *slog.Logger) *AccountProjectionHandler {
	return &AccountProjectionHandler{projector: projector, logger: logger}
}

func (h *AccountProjectionHandler) Handle(ctx context.Context, msg kafka.Message) error {
	switch msg.Topic {
	case domain.AccountCreatedEventType,
		domain.AccountDepositedEventType,
		domain.AccountWithdrawnEventType,
		domain.AccountFrozenEventType,
		domain.AccountUnfrozenEventType,
		domain.AccountDeductedEventType:
		var payload struct {
			AccountID string `json:"account_id"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal account event", "error", err)
			return err
		}
		if payload.AccountID == "" {
			return nil
		}
		return h.projector.Refresh(ctx, payload.AccountID)
	default:
		h.logger.WarnContext(ctx, "unknown account event topic", "topic", msg.Topic)
		return nil
	}
}
