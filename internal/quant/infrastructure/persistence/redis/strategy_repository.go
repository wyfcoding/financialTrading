package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
)

type StrategyRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewStrategyRedisRepository(client redis.UniversalClient) *StrategyRedisRepository {
	return &StrategyRedisRepository{
		client: client,
		prefix: "quant:strategy:",
		ttl:    24 * time.Hour,
	}
}

func (r *StrategyRedisRepository) Save(ctx context.Context, strategy *domain.Strategy) error {
	if strategy == nil {
		return nil
	}
	data, err := json.Marshal(strategy)
	if err != nil {
		return fmt.Errorf("failed to marshal strategy: %w", err)
	}
	return r.client.Set(ctx, r.key(strategy.ID), data, r.ttl).Err()
}

func (r *StrategyRedisRepository) Get(ctx context.Context, id string) (*domain.Strategy, error) {
	if id == "" {
		return nil, nil
	}
	data, err := r.client.Get(ctx, r.key(id)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy from redis: %w", err)
	}
	var strategy domain.Strategy
	if err := json.Unmarshal(data, &strategy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal strategy: %w", err)
	}
	return &strategy, nil
}

func (r *StrategyRedisRepository) key(id string) string {
	return fmt.Sprintf("%s%s", r.prefix, id)
}
