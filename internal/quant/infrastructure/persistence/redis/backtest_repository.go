package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
)

type BacktestRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewBacktestRedisRepository(client redis.UniversalClient) *BacktestRedisRepository {
	return &BacktestRedisRepository{
		client: client,
		prefix: "quant:backtest:",
		ttl:    24 * time.Hour,
	}
}

func (r *BacktestRedisRepository) Save(ctx context.Context, result *domain.BacktestResult) error {
	if result == nil {
		return nil
	}
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal backtest result: %w", err)
	}
	return r.client.Set(ctx, r.key(result.ID), data, r.ttl).Err()
}

func (r *BacktestRedisRepository) Get(ctx context.Context, id string) (*domain.BacktestResult, error) {
	if id == "" {
		return nil, nil
	}
	data, err := r.client.Get(ctx, r.key(id)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get backtest result from redis: %w", err)
	}
	var result domain.BacktestResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backtest result: %w", err)
	}
	return &result, nil
}

func (r *BacktestRedisRepository) key(id string) string {
	return fmt.Sprintf("%s%s", r.prefix, id)
}
