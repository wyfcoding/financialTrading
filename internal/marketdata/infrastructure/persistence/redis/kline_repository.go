package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

type KlineRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
	maxLen int64
}

// NewKlineRedisRepository 创建一个新的基于 Redis 的 K 线读模型仓储。
func NewKlineRedisRepository(client redis.UniversalClient) *KlineRedisRepository {
	return &KlineRedisRepository{
		client: client,
		prefix: "marketdata:kline:",
		ttl:    24 * time.Hour,
		maxLen: 500,
	}
}

func (r *KlineRedisRepository) Save(ctx context.Context, kline *domain.Kline) error {
	if kline == nil {
		return nil
	}
	key := r.prefix + kline.Symbol + ":" + kline.Interval
	data, err := json.Marshal(kline)
	if err != nil {
		return fmt.Errorf("failed to marshal kline: %w", err)
	}
	pipe := r.client.TxPipeline()
	pipe.LPush(ctx, key, data)
	pipe.LTrim(ctx, key, 0, r.maxLen-1)
	pipe.Expire(ctx, key, r.ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *KlineRedisRepository) GetLatest(ctx context.Context, symbol, interval string) (*domain.Kline, error) {
	key := r.prefix + symbol + ":" + interval
	data, err := r.client.LIndex(ctx, key, 0).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get kline from redis: %w", err)
	}
	var kline domain.Kline
	if err := json.Unmarshal(data, &kline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kline: %w", err)
	}
	return &kline, nil
}

func (r *KlineRedisRepository) List(ctx context.Context, symbol, interval string, limit int) ([]*domain.Kline, error) {
	key := r.prefix + symbol + ":" + interval
	if limit <= 0 {
		limit = int(r.maxLen)
	}
	values, err := r.client.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list klines from redis: %w", err)
	}
	klines := make([]*domain.Kline, 0, len(values))
	for _, val := range values {
		var kline domain.Kline
		if err := json.Unmarshal([]byte(val), &kline); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kline: %w", err)
		}
		klines = append(klines, &kline)
	}
	return klines, nil
}
