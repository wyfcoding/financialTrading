package domain

import "context"

// SettlementRepository 结算仓储接口
type SettlementRepository interface {
	Save(ctx context.Context, settlement *Settlement) error
	Get(ctx context.Context, id string) (*Settlement, error)
	GetByTradeID(ctx context.Context, tradeID string) (*Settlement, error)
	List(ctx context.Context, limit int) ([]*Settlement, error)
}

// SettlementSearchRepository 提供基于 Elasticsearch 的结算历史搜索
type SettlementSearchRepository interface {
	Index(ctx context.Context, settlement *Settlement) error
	Search(ctx context.Context, userID, symbol string, limit, offset int) ([]*Settlement, int64, error)
	Delete(ctx context.Context, id string) error
}

// MarginRedisRepository 提供基于 Redis 的实时保证金/风险数据缓存
type MarginRedisRepository interface {
	Save(ctx context.Context, userID string, marginData any) error
	Get(ctx context.Context, userID string) (any, error)
	Delete(ctx context.Context, userID string) error
}
