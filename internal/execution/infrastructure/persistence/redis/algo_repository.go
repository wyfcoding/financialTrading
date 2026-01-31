package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

type algoRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewAlgoRedisRepository(client redis.UniversalClient) domain.AlgoRedisRepository {
	return &algoRedisRepository{
		client: client,
		prefix: "execution:algo:",
		ttl:    24 * time.Hour,
	}
}

func (r *algoRedisRepository) Save(ctx context.Context, order *domain.AlgoOrder) error {
	key := r.prefix + order.AlgoID
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *algoRedisRepository) Get(ctx context.Context, algoID string) (*domain.AlgoOrder, error) {
	key := r.prefix + algoID
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var order domain.AlgoOrder
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *algoRedisRepository) Delete(ctx context.Context, algoID string) error {
	key := r.prefix + algoID
	return r.client.Del(ctx, key).Err()
}
