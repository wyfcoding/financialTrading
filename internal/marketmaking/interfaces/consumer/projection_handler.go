package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/application"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

type ProjectionHandler struct {
	projector *application.MarketMakingProjectionService
	logger    *slog.Logger
}

func NewProjectionHandler(projector *application.MarketMakingProjectionService, logger *slog.Logger) *ProjectionHandler {
	return &ProjectionHandler{projector: projector, logger: logger}
}

func (h *ProjectionHandler) Handle(ctx context.Context, msg kafka.Message) error {
	switch msg.Topic {
	case domain.StrategyCreatedEventType,
		domain.StrategyUpdatedEventType,
		domain.StrategyActivatedEventType,
		domain.StrategyPausedEventType:
		var payload struct {
			Symbol string `json:"symbol"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal strategy event", "error", err)
			return err
		}
		return h.projector.RefreshStrategy(ctx, payload.Symbol, true)
	case domain.PerformanceUpdatedEventType:
		var payload struct {
			Symbol string `json:"symbol"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal performance event", "error", err)
			return err
		}
		return h.projector.RefreshPerformance(ctx, payload.Symbol, true)
	default:
		h.logger.WarnContext(ctx, "unknown marketmaking event topic", "topic", msg.Topic)
		return nil
	}
}
