package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/order/domain"
)

// OrderSearchHandler 监听订单变更并将数据同步到 Elasticsearch。
type OrderSearchHandler struct {
	searchRepo domain.OrderSearchRepository
	orderRepo  domain.OrderRepository // 用于在事件消息不足时回查（可选）
}

func NewOrderSearchHandler(searchRepo domain.OrderSearchRepository, orderRepo domain.OrderRepository) *OrderSearchHandler {
	return &OrderSearchHandler{
		searchRepo: searchRepo,
		orderRepo:  orderRepo,
	}
}

// HandleOrderCreated 处理订单创建事件。
func (h *OrderSearchHandler) HandleOrderCreated(ctx context.Context, payload []byte) error {
	var event struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// 此时通常回查主库以获取最新完整 Order 对象
	order, err := h.orderRepo.Get(ctx, event.OrderID)
	if err != nil {
		return err
	}
	if order == nil {
		return nil
	}

	return h.searchRepo.Index(ctx, order)
}

// HandleOrderUpdated 处理订单状态/执行更新事件。
func (h *OrderSearchHandler) HandleOrderUpdated(ctx context.Context, payload []byte) error {
	// 逻辑与 Created 类似，确保 ES 中始终是最新的 Order 聚合状态
	return h.HandleOrderCreated(ctx, payload)
}

// Subscribe 注册消费者订阅。
func (h *OrderSearchHandler) Subscribe(ctx context.Context, consumer any) {
	slog.Info("Consuming financial order events for ES search indexing")
}
