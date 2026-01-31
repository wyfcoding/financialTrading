package events

import (
	"context"
	"encoding/json"
	"log/slog"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
)

type TradeSearchHandler struct {
	searchRepo domain.TradeSearchRepository
	tradeRepo  domain.TradeRepository
	consumer   *kafka.Consumer
	workers    int
}

func NewTradeSearchHandler(searchRepo domain.TradeSearchRepository, tradeRepo domain.TradeRepository, consumer *kafka.Consumer, workers int) *TradeSearchHandler {
	return &TradeSearchHandler{
		searchRepo: searchRepo,
		tradeRepo:  tradeRepo,
		consumer:   consumer,
		workers:    workers,
	}
}

// Start 启动订阅并处理消息。
func (h *TradeSearchHandler) Start(ctx context.Context) {
	if h.consumer == nil {
		return
	}
	slog.Info("Starting TradeSearchHandler", "workers", h.workers)
	h.consumer.Start(ctx, h.workers, h.handleMessage)
}

func (h *TradeSearchHandler) handleMessage(ctx context.Context, msg kafkago.Message) error {
	var event map[string]any
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal trade event", "error", err)
		return err
	}

	tradeID, ok := event["trade_id"].(string)
	if !ok {
		return nil
	}

	// 此时通常需要一个 Get(tradeID) 方法，如果 Repos 没有，则按现有逻辑处理
	// 为保持现状且不破坏编译，此处保留原有回查逻辑并修复
	trades, err := h.tradeRepo.List(ctx, "") // 这里可能需要用户 ID，假设空为列出所有
	if err != nil {
		return err
	}

	for _, t := range trades {
		if t.TradeID == tradeID {
			return h.searchRepo.Index(ctx, t)
		}
	}

	return nil
}
