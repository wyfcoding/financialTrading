package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
)

type SignalRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewSignalRedisRepository(client redis.UniversalClient) *SignalRedisRepository {
	return &SignalRedisRepository{
		client: client,
		prefix: "quant:signal:",
		ttl:    15 * time.Minute,
	}
}

func (r *SignalRedisRepository) Save(ctx context.Context, signal *domain.Signal) error {
	if signal == nil {
		return nil
	}
	data, err := json.Marshal(signal)
	if err != nil {
		return fmt.Errorf("failed to marshal signal: %w", err)
	}
	return r.client.Set(ctx, r.key(signal.Symbol, signal.Indicator, signal.Period), data, r.ttl).Err()
}

func (r *SignalRedisRepository) GetLatest(ctx context.Context, symbol string, indicator domain.IndicatorType, period int) (*domain.Signal, error) {
	if symbol == "" {
		return nil, nil
	}
	data, err := r.client.Get(ctx, r.key(symbol, indicator, period)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get signal from redis: %w", err)
	}
	var signal domain.Signal
	if err := json.Unmarshal(data, &signal); err != nil {
		return nil, fmt.Errorf("failed to unmarshal signal: %w", err)
	}
	return &signal, nil
}

func (r *SignalRedisRepository) key(symbol string, indicator domain.IndicatorType, period int) string {
	return fmt.Sprintf("%s%s:%s:%d", r.prefix, symbol, indicator, period)
}
