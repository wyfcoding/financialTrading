package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/wyfcoding/financialtrading/internal/risk/application"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

type ProjectionHandler struct {
	projector *application.RiskProjectionService
	logger    *slog.Logger
}

func NewProjectionHandler(projector *application.RiskProjectionService, logger *slog.Logger) *ProjectionHandler {
	return &ProjectionHandler{projector: projector, logger: logger}
}

func (h *ProjectionHandler) Handle(ctx context.Context, msg kafka.Message) error {
	switch msg.Topic {
	case domain.RiskAssessmentCreatedEventType:
		var payload struct {
			AssessmentID string `json:"assessment_id"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal assessment event", "error", err)
			return err
		}
		return h.projector.RefreshAssessment(ctx, payload.AssessmentID, true)
	case domain.RiskAlertGeneratedEventType:
		var payload struct {
			AlertID string `json:"alert_id"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal alert event", "error", err)
			return err
		}
		return h.projector.RefreshAlert(ctx, payload.AlertID, true)
	case domain.RiskLimitUpdatedEventType, domain.RiskLimitExceededEventType:
		var payload struct {
			UserID    string `json:"user_id"`
			LimitType string `json:"limit_type"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal limit event", "error", err)
			return err
		}
		return h.projector.RefreshRiskLimit(ctx, payload.UserID, payload.LimitType)
	case domain.RiskMetricsUpdatedEventType:
		var payload struct {
			UserID string `json:"user_id"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal metrics event", "error", err)
			return err
		}
		return h.projector.RefreshRiskMetrics(ctx, payload.UserID)
	case domain.CircuitBreakerFiredEventType, domain.CircuitBreakerResetEventType:
		var payload struct {
			UserID string `json:"user_id"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal circuit breaker event", "error", err)
			return err
		}
		return h.projector.RefreshCircuitBreaker(ctx, payload.UserID)
	default:
		h.logger.WarnContext(ctx, "unknown risk event topic", "topic", msg.Topic)
		return nil
	}
}
