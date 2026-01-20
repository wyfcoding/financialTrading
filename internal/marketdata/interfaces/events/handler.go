package events

import (
	"context"
	"encoding/json"
	"log/slog"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/application"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
)

type MarketDataEventHandler struct {
	service *application.MarketDataService
}

func NewMarketDataEventHandler(service *application.MarketDataService) *MarketDataEventHandler {
	return &MarketDataEventHandler{service: service}
}

func (h *MarketDataEventHandler) HandleMarketPrice(ctx context.Context, msg kafkago.Message) error {
	var event struct {
		Symbol    string `json:"symbol"`
		Price     string `json:"price"`
		Timestamp int64  `json:"timestamp"`
	}
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}

	price, _ := decimal.NewFromString(event.Price)
	slog.Info("Handling market price event", "symbol", event.Symbol, "price", price.String())

	return h.service.SaveQuote(ctx, event.Symbol, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, price, decimal.Zero, event.Timestamp, "simulation")
}

func (h *MarketDataEventHandler) Subscribe(ctx context.Context, consumer *kafka.Consumer) {
	consumer.Start(ctx, 1, h.HandleMarketPrice)
}
