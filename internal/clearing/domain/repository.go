package domain

import "context"

// SettlementRepository 结算仓储接口
type SettlementRepository interface {
	Save(ctx context.Context, settlement *Settlement) error
	Get(ctx context.Context, id string) (*Settlement, error)
	GetByTradeID(ctx context.Context, tradeID string) (*Settlement, error)
}
