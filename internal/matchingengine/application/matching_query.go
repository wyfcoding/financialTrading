package application

import (
	"context"

	"github.com/wyfcoding/pkg/algorithm/types"

	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
)

// MatchingEngineQuery 处理所有撮合引擎相关的查询操作（Queries）。
type MatchingEngineQuery struct {
	engine    *domain.DisruptionEngine
	tradeRepo domain.TradeRepository
}

// NewMatchingEngineQuery 构造函数。
func NewMatchingEngineQuery(engine *domain.DisruptionEngine, tradeRepo domain.TradeRepository) *MatchingEngineQuery {
	return &MatchingEngineQuery{
		engine:    engine,
		tradeRepo: tradeRepo,
	}
}

// GetOrderBook 获取订单簿快照
func (q *MatchingEngineQuery) GetOrderBook(ctx context.Context, depth int) (*domain.OrderBookSnapshot, error) {
	if depth <= 0 {
		depth = 20
	}
	return q.engine.GetOrderBookSnapshot(depth), nil
}

// GetTrades 获取成交历史
func (q *MatchingEngineQuery) GetTrades(ctx context.Context, symbol string, limit int) ([]*types.Trade, error) {
	if limit <= 0 {
		limit = 100
	}
	return q.tradeRepo.GetLatestTrades(ctx, symbol, limit)
}
