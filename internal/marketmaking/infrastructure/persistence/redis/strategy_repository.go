package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

type StrategyRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewStrategyRedisRepository(client redis.UniversalClient) *StrategyRedisRepository {
	return &StrategyRedisRepository{
		client: client,
		prefix: "marketmaking:strategy:",
		ttl:    24 * time.Hour,
	}
}

func (r *StrategyRedisRepository) Save(ctx context.Context, strategy *domain.QuoteStrategy) error {
	if strategy == nil {
		return nil
	}
	key := r.prefix + strategy.Symbol
	data, err := json.Marshal(strategy)
	if err != nil {
		return fmt.Errorf("failed to marshal strategy: %w", err)
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *StrategyRedisRepository) Get(ctx context.Context, symbol string) (*domain.QuoteStrategy, error) {
	key := r.prefix + symbol
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get strategy from redis: %w", err)
	}

	var strategy domain.QuoteStrategy
	if err := json.Unmarshal(data, &strategy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal strategy: %w", err)
	}
	return &strategy, nil
}
