package domain

import (
	"context"
)

// TradeRepository 成交记录仓储接口
type TradeRepository interface {
	Save(ctx context.Context, trade *Trade) error
	GetLatestTrades(ctx context.Context, symbol string, limit int) ([]*Trade, error)
}

// OrderBookRepository 订单簿仓储接口
type OrderBookRepository interface {
	SaveSnapshot(ctx context.Context, snapshot *OrderBookSnapshot) error
}
