package domain

import "context"

// TradeRepository 成交单仓储接口
type TradeRepository interface {
	Save(ctx context.Context, trade *Trade) error
	GetByOrderID(ctx context.Context, orderID string) (*Trade, error)
	List(ctx context.Context, userID string) ([]*Trade, error)
}

// AlgoOrderRepository 算法订单仓储接口
type AlgoOrderRepository interface {
	Save(ctx context.Context, order *AlgoOrder) error
	Get(ctx context.Context, algoID string) (*AlgoOrder, error)
	ListActive(ctx context.Context) ([]*AlgoOrder, error)
}
