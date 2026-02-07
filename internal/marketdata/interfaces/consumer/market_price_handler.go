package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/application"
)

type MarketDataEventHandler struct {
	command *application.MarketDataCommandService
}

func NewMarketDataEventHandler(command *application.MarketDataCommandService) *MarketDataEventHandler {
	return &MarketDataEventHandler{command: command}
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

	return h.command.SaveQuote(ctx, application.SaveQuoteCommand{
		Symbol:    event.Symbol,
		LastPrice: price,
		Timestamp: event.Timestamp,
		Source:    "simulation",
	})
}
