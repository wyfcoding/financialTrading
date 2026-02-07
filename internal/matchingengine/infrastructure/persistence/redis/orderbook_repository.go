package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
)

type OrderBookRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewOrderBookRedisRepository(client redis.UniversalClient) *OrderBookRedisRepository {
	return &OrderBookRedisRepository{
		client: client,
		prefix: "matching:orderbook:",
		ttl:    2 * time.Second,
	}
}

func (r *OrderBookRedisRepository) Save(ctx context.Context, snapshot *domain.OrderBookSnapshot, depth int) error {
	if snapshot == nil {
		return nil
	}
	key := r.key(snapshot.Symbol, depth)
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal orderbook snapshot: %w", err)
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *OrderBookRedisRepository) Get(ctx context.Context, symbol string, depth int) (*domain.OrderBookSnapshot, error) {
	key := r.key(symbol, depth)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get orderbook snapshot from redis: %w", err)
	}
	var snapshot domain.OrderBookSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal orderbook snapshot: %w", err)
	}
	return &snapshot, nil
}

func (r *OrderBookRedisRepository) key(symbol string, depth int) string {
	if depth <= 0 {
		depth = 0
	}
	return fmt.Sprintf("%s%s:%d", r.prefix, symbol, depth)
}
