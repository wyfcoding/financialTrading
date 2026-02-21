//go:build matching_experimental
// +build matching_experimental

package application

import (
	"context"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain/orderbook"
)

type MatchingService struct {
	engine *domain.MatchingEngine
}

func (s *MatchingService) HandleOrder(ctx context.Context, order *orderbook.Order) ([]*orderbook.Trade, error) {
	// 逻辑：接收 -> 风控(略) -> 撮合
	return s.engine.Process(order)
}
