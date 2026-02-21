// Package consumer 处理外部事件触发结算
package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/wyfcoding/financialtrading/internal/settlement/application"
)

// MatchedEventConsumer 撮合完成事件消费者
type MatchedEventConsumer struct {
	settleApp *application.SettlementAppService
	logger    *slog.Logger
}

func NewMatchedEventConsumer(app *application.SettlementAppService, logger *slog.Logger) *MatchedEventConsumer {
	return &MatchedEventConsumer{
		settleApp: app,
		logger:    logger.With("module", "matched_consumer"),
	}
}

// Handle 处理撮合结果
func (c *MatchedEventConsumer) Handle(ctx context.Context, msg kafka.Message) error {
	var event struct {
		TradeID     string  `json:"trade_id"`
		BuyOrderID  string  `json:"buy_order_id"`
		SellOrderID string  `json:"sell_order_id"`
		Symbol      string  `json:"symbol"`
		Price       float64 `json:"price"`
		Quantity    float64 `json:"quantity"`
		MatchedAt   int64   `json:"matched_at"`
	}

	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.logger.Error("failed to unmarshal matched event", "error", err)
		return err
	}

	c.logger.Info("received matched event, initiating settlement", "trade_id", event.TradeID)

	ins, err := c.settleApp.CreateInstruction(ctx, application.CreateInstructionCommand{
		TradeID:         event.TradeID,
		OrderID:         event.BuyOrderID,
		Symbol:          event.Symbol,
		Quantity:        event.Quantity,
		Price:           event.Price,
		Currency:        "USD",
		BuyerAccountID:  event.BuyOrderID,
		SellerAccountID: event.SellOrderID,
		CycleDays:       2,
	})
	if err != nil {
		c.logger.Error("failed to create settlement instruction", "trade_id", event.TradeID, "error", err)
		return err
	}

	return c.settleApp.ProcessSettlement(ctx, application.ProcessSettlementCommand{
		InstructionID: ins.InstructionID,
	})
}
