package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
)

// MatchingQueryService 处理所有撮合引擎相关的查询操作（Queries）。
type MatchingQueryService struct {
	engine    *domain.DisruptionEngine
	tradeRepo domain.TradeRepository
}

// NewMatchingQueryService 构造函数。
func NewMatchingQueryService(engine *domain.DisruptionEngine, tradeRepo domain.TradeRepository) *MatchingQueryService {
	return &MatchingQueryService{
		engine:    engine,
		tradeRepo: tradeRepo,
	}
}

// GetOrderBook 获取订单簿快照
func (q *MatchingQueryService) GetOrderBook(ctx context.Context, depth int) (*domain.OrderBookSnapshot, error) {
	if depth <= 0 {
		depth = 20
	}
	return q.engine.GetOrderBookSnapshot(depth), nil
}

// GetTrades 获取成交历史
func (q *MatchingQueryService) GetTrades(ctx context.Context, symbol string, limit int) ([]*domain.Trade, error) {
	if limit <= 0 {
		limit = 100
	}
	return q.tradeRepo.GetLatestTrades(ctx, symbol, limit)
}
