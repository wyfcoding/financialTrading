package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
)

type marginRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewMarginRedisRepository(client redis.UniversalClient) domain.MarginRedisRepository {
	return &marginRedisRepository{
		client: client,
		prefix: "clearing:margin:",
		ttl:    1 * time.Hour,
	}
}

func (r *marginRedisRepository) Save(ctx context.Context, userID string, marginData any) error {
	key := r.prefix + userID
	data, err := json.Marshal(marginData)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *marginRedisRepository) Get(ctx context.Context, userID string) (any, error) {
	key := r.prefix + userID
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var res any
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (r *marginRedisRepository) Delete(ctx context.Context, userID string) error {
	key := r.prefix + userID
	return r.client.Del(ctx, key).Err()
}
