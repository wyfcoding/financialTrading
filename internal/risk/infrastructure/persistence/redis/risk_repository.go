package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

type riskRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewRiskRedisRepository(client redis.UniversalClient) domain.RiskRedisRepository {
	return &riskRedisRepository{
		client: client,
		prefix: "risk:",
		ttl:    1 * time.Hour,
	}
}

func (r *riskRedisRepository) SaveLimit(ctx context.Context, userID string, limit *domain.RiskLimit) error {
	key := fmt.Sprintf("%slimit:%s:%s", r.prefix, userID, limit.LimitType)
	data, err := json.Marshal(limit)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *riskRedisRepository) GetLimit(ctx context.Context, userID, limitType string) (*domain.RiskLimit, error) {
	key := fmt.Sprintf("%slimit:%s:%s", r.prefix, userID, limitType)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var limit domain.RiskLimit
	if err := json.Unmarshal(data, &limit); err != nil {
		return nil, err
	}
	return &limit, nil
}

func (r *riskRedisRepository) DeleteLimit(ctx context.Context, userID, limitType string) error {
	key := fmt.Sprintf("%slimit:%s:%s", r.prefix, userID, limitType)
	return r.client.Del(ctx, key).Err()
}

func (r *riskRedisRepository) SaveMetrics(ctx context.Context, userID string, metrics *domain.RiskMetrics) error {
	key := fmt.Sprintf("%smetrics:%s", r.prefix, userID)
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *riskRedisRepository) GetMetrics(ctx context.Context, userID string) (*domain.RiskMetrics, error) {
	key := fmt.Sprintf("%smetrics:%s", r.prefix, userID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var metrics domain.RiskMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, err
	}
	return &metrics, nil
}

func (r *riskRedisRepository) DeleteMetrics(ctx context.Context, userID string) error {
	key := fmt.Sprintf("%smetrics:%s", r.prefix, userID)
	return r.client.Del(ctx, key).Err()
}

func (r *riskRedisRepository) SaveCircuitBreaker(ctx context.Context, userID string, cb *domain.CircuitBreaker) error {
	key := fmt.Sprintf("%scb:%s", r.prefix, userID)
	data, err := json.Marshal(cb)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *riskRedisRepository) GetCircuitBreaker(ctx context.Context, userID string) (*domain.CircuitBreaker, error) {
	key := fmt.Sprintf("%scb:%s", r.prefix, userID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var cb domain.CircuitBreaker
	if err := json.Unmarshal(data, &cb); err != nil {
		return nil, err
	}
	return &cb, nil
}

func (r *riskRedisRepository) DeleteCircuitBreaker(ctx context.Context, userID string) error {
	key := fmt.Sprintf("%scb:%s", r.prefix, userID)
	return r.client.Del(ctx, key).Err()
}
