// 变更说明：实现撮合引擎的订单流消费者。
// 承担“网关 -> 处理”的关键链路，将消息队列中的新订单喂入撮合队列。
package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/matchingengine/application"
)

type OrderStreamConsumer struct {
	svc    *application.MatchingCommandService
	logger *slog.Logger
}

func (c *OrderStreamConsumer) ConsumeOrder(ctx context.Context, data []byte) error {
	var cmd application.SubmitOrderCommand
	if err := json.Unmarshal(data, &cmd); err != nil {
		c.logger.Error("failed to decode order command", "error", err)
		return err
	}

	// 提交到撮合引擎内存队列
	_, err := c.svc.SubmitOrder(ctx, &cmd)
	return err
}
