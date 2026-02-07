package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

type PerformanceRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewPerformanceRedisRepository(client redis.UniversalClient) *PerformanceRedisRepository {
	return &PerformanceRedisRepository{
		client: client,
		prefix: "marketmaking:performance:",
		ttl:    24 * time.Hour,
	}
}

func (r *PerformanceRedisRepository) Save(ctx context.Context, performance *domain.MarketMakingPerformance) error {
	if performance == nil {
		return nil
	}
	key := r.prefix + performance.Symbol
	data, err := json.Marshal(performance)
	if err != nil {
		return fmt.Errorf("failed to marshal performance: %w", err)
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *PerformanceRedisRepository) Get(ctx context.Context, symbol string) (*domain.MarketMakingPerformance, error) {
	key := r.prefix + symbol
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get performance from redis: %w", err)
	}

	var performance domain.MarketMakingPerformance
	if err := json.Unmarshal(data, &performance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal performance: %w", err)
	}
	return &performance, nil
}
