package domain

import (
	"context"

	"github.com/wyfcoding/pkg/algorithm/types"
)

// TradeRepository 成交记录仓储接口
type TradeRepository interface {
	Save(ctx context.Context, trade *types.Trade) error
	GetLatestTrades(ctx context.Context, symbol string, limit int) ([]*types.Trade, error)
}

// OrderBookRepository 订单簿仓储接口
type OrderBookRepository interface {
	SaveSnapshot(ctx context.Context, snapshot *OrderBookSnapshot) error
}
