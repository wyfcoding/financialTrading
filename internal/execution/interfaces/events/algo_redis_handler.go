package events

import (
	"context"
	"encoding/json"

	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

type AlgoRedisHandler struct {
	redisRepo domain.AlgoRedisRepository
	algoRepo  domain.AlgoOrderRepository
}

func NewAlgoRedisHandler(redisRepo domain.AlgoRedisRepository, algoRepo domain.AlgoOrderRepository) *AlgoRedisHandler {
	return &AlgoRedisHandler{
		redisRepo: redisRepo,
		algoRepo:  algoRepo,
	}
}

func (h *AlgoRedisHandler) OnAlgoUpdated(ctx context.Context, payload []byte) error {
	var event struct {
		AlgoID string `json:"algo_id"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	order, err := h.algoRepo.Get(ctx, event.AlgoID)
	if err != nil {
		return err
	}
	if order == nil {
		return nil
	}

	return h.redisRepo.Save(ctx, order)
}
