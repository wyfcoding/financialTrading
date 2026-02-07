package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/wyfcoding/financialtrading/internal/execution/application"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

type TradeProjectionHandler struct {
	projector *application.ExecutionProjectionService
	logger    *slog.Logger
}

func NewTradeProjectionHandler(projector *application.ExecutionProjectionService, logger *slog.Logger) *TradeProjectionHandler {
	return &TradeProjectionHandler{projector: projector, logger: logger}
}

func (h *TradeProjectionHandler) Handle(ctx context.Context, msg kafka.Message) error {
	if msg.Topic != domain.TradeExecutedEventType {
		h.logger.WarnContext(ctx, "unknown execution trade event topic", "topic", msg.Topic)
		return nil
	}
	var payload struct {
		TradeID string `json:"trade_id"`
	}
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		h.logger.ErrorContext(ctx, "failed to unmarshal trade event", "error", err)
		return err
	}
	if payload.TradeID == "" {
		return nil
	}
	return h.projector.RefreshTrade(ctx, payload.TradeID, true)
}
