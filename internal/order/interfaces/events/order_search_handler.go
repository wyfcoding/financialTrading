package events

import (
	"context"
	"encoding/json"
	"log/slog"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
)

// OrderSearchHandler 监听订单变更并将数据同步到 Elasticsearch。
type OrderSearchHandler struct {
	searchRepo domain.OrderSearchRepository
	orderRepo  domain.OrderRepository
	consumer   *kafka.Consumer
	workers    int
}

func NewOrderSearchHandler(searchRepo domain.OrderSearchRepository, orderRepo domain.OrderRepository, consumer *kafka.Consumer, workers int) *OrderSearchHandler {
	return &OrderSearchHandler{
		searchRepo: searchRepo,
		orderRepo:  orderRepo,
		consumer:   consumer,
		workers:    workers,
	}
}

// Start 启动订阅并处理消息。
func (h *OrderSearchHandler) Start(ctx context.Context) {
	if h.consumer == nil {
		return
	}
	slog.Info("Starting OrderSearchHandler", "workers", h.workers)
	h.consumer.Start(ctx, h.workers, h.handleMessage)
}

func (h *OrderSearchHandler) handleMessage(ctx context.Context, msg kafkago.Message) error {
	var event map[string]any
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal order event", "error", err)
		return err
	}

	orderID, ok := event["order_id"].(string)
	if !ok || orderID == "" {
		return nil
	}

	order, err := h.orderRepo.Get(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return nil
	}

	return h.searchRepo.Index(ctx, order)
}
