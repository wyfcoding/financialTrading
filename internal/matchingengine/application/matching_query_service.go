package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
)

// MatchingQueryService 处理所有撮合引擎相关的查询操作（Queries）。
type MatchingQueryService struct {
	engine          *domain.DisruptionEngine
	tradeRepo       domain.TradeRepository
	tradeReadRepo   domain.TradeReadRepository
	tradeSearchRepo domain.TradeSearchRepository
	orderBookReadRepo domain.OrderBookReadRepository
}

// NewMatchingQueryService 构造函数。
func NewMatchingQueryService(
	engine *domain.DisruptionEngine,
	tradeRepo domain.TradeRepository,
	tradeReadRepo domain.TradeReadRepository,
	tradeSearchRepo domain.TradeSearchRepository,
	orderBookReadRepo domain.OrderBookReadRepository,
) *MatchingQueryService {
	return &MatchingQueryService{
		engine:           engine,
		tradeRepo:        tradeRepo,
		tradeReadRepo:    tradeReadRepo,
		tradeSearchRepo:  tradeSearchRepo,
		orderBookReadRepo: orderBookReadRepo,
	}
}

// GetOrderBook 获取订单簿快照
func (q *MatchingQueryService) GetOrderBook(ctx context.Context, depth int) (*domain.OrderBookSnapshot, error) {
	if depth <= 0 {
		depth = 20
	}
	if q.orderBookReadRepo != nil {
		if cached, err := q.orderBookReadRepo.Get(ctx, q.engine.Symbol(), depth); err == nil && cached != nil {
			return cached, nil
		}
	}

	snapshot := q.engine.GetOrderBookSnapshot(depth)
	if q.orderBookReadRepo != nil {
		_ = q.orderBookReadRepo.Save(ctx, snapshot, depth)
	}
	return snapshot, nil
}

// GetTrades 获取成交历史
func (q *MatchingQueryService) GetTrades(ctx context.Context, symbol string, limit int) ([]*domain.Trade, error) {
	if limit <= 0 {
		limit = 100
	}
	if q.tradeReadRepo != nil {
		if cached, err := q.tradeReadRepo.List(ctx, symbol, limit); err == nil && len(cached) > 0 {
			return cached, nil
		}
	}
	if q.tradeSearchRepo != nil {
		if trades, _, err := q.tradeSearchRepo.Search(ctx, symbol, limit, 0); err == nil && len(trades) > 0 {
			return trades, nil
		}
	}
	return q.tradeRepo.GetLatestTrades(ctx, symbol, limit)
}
