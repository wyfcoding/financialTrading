package events

import (
	"context"
	"encoding/json"

	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

type TradeSearchHandler struct {
	searchRepo domain.TradeSearchRepository
	tradeRepo  domain.TradeRepository
}

func NewTradeSearchHandler(searchRepo domain.TradeSearchRepository, tradeRepo domain.TradeRepository) *TradeSearchHandler {
	return &TradeSearchHandler{
		searchRepo: searchRepo,
		tradeRepo:  tradeRepo,
	}
}

func (h *TradeSearchHandler) OnTradeExecuted(ctx context.Context, payload []byte) error {
	var event struct {
		TradeID string `json:"trade_id"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// 从主库获取最新状态并同步到 ES
	// 注意：订单 ID 的 trade 需要通过 Repo 获取，或者 Payload 已经包含全量信息
	// 这里为了简单，假设 Payload 已包含全量信息，或者重新从 DB 拉取。
	// 标准的做法是 Payload 包含关键 ID，Handler 向 DB 拉取最新 Aggregate。

	// 这里先简单处理，假设我们要从 DB 拉取
	trades, err := h.tradeRepo.List(ctx, "") // 实际上需要按 TradeID 获取，此处 domain 接口需增强
	if err != nil {
		return err
	}

	for _, t := range trades {
		if t.TradeID == event.TradeID {
			return h.searchRepo.Index(ctx, t)
		}
	}

	return nil
}
