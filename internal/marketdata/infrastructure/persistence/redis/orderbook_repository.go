package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

type OrderBookRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

// NewOrderBookRedisRepository 创建一个新的基于 Redis 的订单簿读模型仓储。
func NewOrderBookRedisRepository(client redis.UniversalClient) *OrderBookRedisRepository {
	return &OrderBookRedisRepository{
		client: client,
		prefix: "marketdata:orderbook:",
		ttl:    5 * time.Minute,
	}
}

func (r *OrderBookRedisRepository) Save(ctx context.Context, ob *domain.OrderBook) error {
	if ob == nil {
		return nil
	}
	key := r.prefix + ob.Symbol
	data, err := json.Marshal(ob)
	if err != nil {
		return fmt.Errorf("failed to marshal orderbook: %w", err)
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *OrderBookRedisRepository) Get(ctx context.Context, symbol string) (*domain.OrderBook, error) {
	key := r.prefix + symbol
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get orderbook from redis: %w", err)
	}
	var ob domain.OrderBook
	if err := json.Unmarshal(data, &ob); err != nil {
		return nil, fmt.Errorf("failed to unmarshal orderbook: %w", err)
	}
	return &ob, nil
}
