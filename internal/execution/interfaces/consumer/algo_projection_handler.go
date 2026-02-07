package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/wyfcoding/financialtrading/internal/execution/application"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

type AlgoProjectionHandler struct {
	projector *application.ExecutionProjectionService
	logger    *slog.Logger
}

func NewAlgoProjectionHandler(projector *application.ExecutionProjectionService, logger *slog.Logger) *AlgoProjectionHandler {
	return &AlgoProjectionHandler{projector: projector, logger: logger}
}

func (h *AlgoProjectionHandler) Handle(ctx context.Context, msg kafka.Message) error {
	if msg.Topic != domain.AlgoOrderStartedEventType {
		h.logger.WarnContext(ctx, "unknown execution algo event topic", "topic", msg.Topic)
		return nil
	}
	var payload struct {
		AlgoID string `json:"algo_id"`
	}
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		h.logger.ErrorContext(ctx, "failed to unmarshal algo event", "error", err)
		return err
	}
	if payload.AlgoID == "" {
		return nil
	}
	return h.projector.RefreshAlgo(ctx, payload.AlgoID)
}
