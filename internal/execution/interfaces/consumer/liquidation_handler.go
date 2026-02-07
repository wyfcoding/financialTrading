package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	risk_pb "github.com/wyfcoding/financialtrading/go-api/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/application"
)

// LiquidationHandler 监听来自风险服务的强平触发事件并执行平仓操作。
type LiquidationHandler struct {
	service *application.ExecutionCommandService
}

func NewLiquidationHandler(service *application.ExecutionCommandService) *LiquidationHandler {
	return &LiquidationHandler{service: service}
}

// Handle 处理 PositionLiquidationTriggeredEvent 事件。
func (h *LiquidationHandler) Handle(ctx context.Context, payload []byte) error {
	var event risk_pb.PositionLiquidationTriggeredEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		slog.Error("failed to unmarshal liquidation event", "error", err)
		return err
	}

	slog.Info("Received liquidation trigger",
		"user_id", event.UserId,
		"symbol", event.Symbol,
		"side", event.Side,
		"quantity", event.Quantity)

	// 执行平仓逻辑
	if err := h.service.HandleLiquidation(ctx, event.UserId, event.Symbol, event.Side, event.Quantity); err != nil {
		slog.Error("failed to handle liquidation", "user_id", event.UserId, "symbol", event.Symbol, "error", err)
		return err
	}

	slog.Info("Liquidation executed successfully", "user_id", event.UserId, "symbol", event.Symbol)
	return nil
}
