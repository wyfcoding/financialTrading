package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/application"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
)

type ProjectionHandler struct {
	projector *application.MatchingProjectionService
	logger    *slog.Logger
}

func NewProjectionHandler(projector *application.MatchingProjectionService, logger *slog.Logger) *ProjectionHandler {
	return &ProjectionHandler{projector: projector, logger: logger}
}

func (h *ProjectionHandler) Handle(ctx context.Context, msg kafka.Message) error {
	switch msg.Topic {
	case domain.TradeExecutedEventType:
		var payload struct {
			TradeID     string `json:"trade_id"`
			BuyOrderID  string `json:"buy_order_id"`
			SellOrderID string `json:"sell_order_id"`
			Symbol      string `json:"symbol"`
			Quantity    string `json:"quantity"`
			Price       string `json:"price"`
			ExecutedAt  int64  `json:"executed_at"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal trade event", "error", err)
			return err
		}
		qty, _ := decimal.NewFromString(payload.Quantity)
		px, _ := decimal.NewFromString(payload.Price)
		trade := &domain.Trade{
			TradeID:     payload.TradeID,
			BuyOrderID:  payload.BuyOrderID,
			SellOrderID: payload.SellOrderID,
			Symbol:      payload.Symbol,
			Price:       px.InexactFloat64(),
			Quantity:    qty.InexactFloat64(),
			Timestamp:   time.Unix(0, payload.ExecutedAt),
		}
		return h.projector.ProjectTrade(ctx, trade)
	default:
		h.logger.WarnContext(ctx, "unknown matching event topic", "topic", msg.Topic)
		return nil
	}
}
