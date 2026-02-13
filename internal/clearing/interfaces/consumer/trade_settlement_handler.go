package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/clearing/application"
)

const matchingTradeExecutedTopic = "matching.trade.executed"

// TradeSettlementHandler 消费撮合成交事件并触发清算。
type TradeSettlementHandler struct {
	cmd    *application.ClearingCommandService
	logger *slog.Logger
}

func NewTradeSettlementHandler(cmd *application.ClearingCommandService, logger *slog.Logger) *TradeSettlementHandler {
	return &TradeSettlementHandler{cmd: cmd, logger: logger}
}

func (h *TradeSettlementHandler) Handle(ctx context.Context, msg kafka.Message) error {
	if msg.Topic != matchingTradeExecutedTopic {
		return nil
	}

	var payload struct {
		TradeID    string `json:"trade_id"`
		BuyUserID  string `json:"buy_user_id"`
		SellUserID string `json:"sell_user_id"`
		Symbol     string `json:"symbol"`
		Quantity   string `json:"quantity"`
		Price      string `json:"price"`
		Currency   string `json:"currency"`
	}

	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		h.logger.ErrorContext(ctx, "failed to unmarshal matching trade event", "error", err)
		return err
	}
	if payload.TradeID == "" {
		return nil
	}

	qty, err := decimal.NewFromString(payload.Quantity)
	if err != nil {
		h.logger.ErrorContext(ctx, "invalid trade quantity", "trade_id", payload.TradeID, "quantity", payload.Quantity, "error", err)
		return err
	}
	price, err := decimal.NewFromString(payload.Price)
	if err != nil {
		h.logger.ErrorContext(ctx, "invalid trade price", "trade_id", payload.TradeID, "price", payload.Price, "error", err)
		return err
	}

	currency := payload.Currency
	if currency == "" {
		currency = inferCurrencyFromSymbol(payload.Symbol)
	}

	_, err = h.cmd.SettleTrade(ctx, &application.SettleTradeRequest{
		TradeID:    payload.TradeID,
		BuyUserID:  payload.BuyUserID,
		SellUserID: payload.SellUserID,
		Symbol:     payload.Symbol,
		Quantity:   qty,
		Price:      price,
		Currency:   currency,
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to settle trade from event", "trade_id", payload.TradeID, "error", err)
		return err
	}
	return nil
}

func inferCurrencyFromSymbol(symbol string) string {
	if symbol == "" {
		return "USDT"
	}
	parts := strings.FieldsFunc(symbol, func(r rune) bool {
		return r == '-' || r == '/' || r == '_'
	})
	if len(parts) >= 2 && parts[len(parts)-1] != "" {
		return strings.ToUpper(parts[len(parts)-1])
	}
	return "USDT"
}
