package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/auth/domain"
)

type apiKeyRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewAPIKeyRedisRepository(client redis.UniversalClient) domain.APIKeyRedisRepository {
	return &apiKeyRedisRepository{
		client: client,
		prefix: "auth:apikey:",
		ttl:    1 * time.Hour,
	}
}

func (r *apiKeyRedisRepository) Save(ctx context.Context, key *domain.APIKey) error {
	redisKey := fmt.Sprintf("%s%s", r.prefix, key.Key)
	data, err := json.Marshal(key)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, redisKey, data, r.ttl).Err()
}

func (r *apiKeyRedisRepository) Get(ctx context.Context, key string) (*domain.APIKey, error) {
	redisKey := fmt.Sprintf("%s%s", r.prefix, key)
	data, err := r.client.Get(ctx, redisKey).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var ak domain.APIKey
	if err := json.Unmarshal(data, &ak); err != nil {
		return nil, err
	}
	return &ak, nil
}

func (r *apiKeyRedisRepository) Delete(ctx context.Context, key string) error {
	redisKey := fmt.Sprintf("%s%s", r.prefix, key)
	return r.client.Del(ctx, redisKey).Err()
}
