package events

import (
	"context"
	"encoding/json"
	"log/slog"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
)

type AlgoRedisHandler struct {
	redisRepo domain.AlgoRedisRepository
	algoRepo  domain.AlgoOrderRepository
	consumer  *kafka.Consumer
	workers   int
}

func NewAlgoRedisHandler(redisRepo domain.AlgoRedisRepository, algoRepo domain.AlgoOrderRepository, consumer *kafka.Consumer, workers int) *AlgoRedisHandler {
	return &AlgoRedisHandler{
		redisRepo: redisRepo,
		algoRepo:  algoRepo,
		consumer:  consumer,
		workers:   workers,
	}
}

// Start 启动订阅并处理消息。
func (h *AlgoRedisHandler) Start(ctx context.Context) {
	if h.consumer == nil {
		return
	}
	slog.Info("Starting AlgoRedisHandler", "workers", h.workers)
	h.consumer.Start(ctx, h.workers, h.handleMessage)
}

func (h *AlgoRedisHandler) handleMessage(ctx context.Context, msg kafkago.Message) error {
	var event map[string]any
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal algo event", "error", err)
		return err
	}

	algoID, ok := event["algo_id"].(string)
	if !ok {
		return nil
	}

	order, err := h.algoRepo.Get(ctx, algoID)
	if err != nil {
		return err
	}
	if order == nil {
		return nil
	}

	return h.redisRepo.Save(ctx, order)
}
