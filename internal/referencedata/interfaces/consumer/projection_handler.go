package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/wyfcoding/financialtrading/internal/referencedata/application"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

type ProjectionHandler struct {
	projector *application.ReferenceDataProjectionService
	logger    *slog.Logger
}

func NewProjectionHandler(projector *application.ReferenceDataProjectionService, logger *slog.Logger) *ProjectionHandler {
	return &ProjectionHandler{projector: projector, logger: logger}
}

func (h *ProjectionHandler) Handle(ctx context.Context, msg kafka.Message) error {
	switch msg.Topic {
	case domain.SymbolCreatedEventType,
		domain.SymbolUpdatedEventType,
		domain.SymbolStatusChangedEventType,
		domain.SymbolDeletedEventType:
		var payload struct {
			SymbolID   string `json:"symbol_id"`
			SymbolCode string `json:"symbol_code"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal symbol event", "error", err)
			return err
		}
		id := payload.SymbolID
		if id == "" {
			id = payload.SymbolCode
		}
		return h.projector.RefreshSymbol(ctx, id, true)
	case domain.ExchangeCreatedEventType,
		domain.ExchangeUpdatedEventType,
		domain.ExchangeStatusChangedEventType,
		domain.ExchangeDeletedEventType:
		var payload struct {
			ExchangeID string `json:"exchange_id"`
			Name       string `json:"name"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal exchange event", "error", err)
			return err
		}
		id := payload.ExchangeID
		if id == "" {
			id = payload.Name
		}
		return h.projector.RefreshExchange(ctx, id, true)
	default:
		h.logger.WarnContext(ctx, "unknown referencedata event topic", "topic", msg.Topic)
		return nil
	}
}
