package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

type riskReadRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewRiskReadRepository(client redis.UniversalClient) domain.RiskReadRepository {
	return &riskReadRepository{
		client: client,
		prefix: "risk:",
		ttl:    1 * time.Hour,
	}
}

func (r *riskReadRepository) SaveLimit(ctx context.Context, userID string, limit *domain.RiskLimit) error {
	if limit == nil {
		return nil
	}
	key := fmt.Sprintf("%slimit:%s:%s", r.prefix, userID, limit.LimitType)
	data, err := json.Marshal(limit)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *riskReadRepository) GetLimit(ctx context.Context, userID, limitType string) (*domain.RiskLimit, error) {
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

func (r *riskReadRepository) DeleteLimit(ctx context.Context, userID, limitType string) error {
	key := fmt.Sprintf("%slimit:%s:%s", r.prefix, userID, limitType)
	return r.client.Del(ctx, key).Err()
}

func (r *riskReadRepository) SaveMetrics(ctx context.Context, userID string, metrics *domain.RiskMetrics) error {
	if metrics == nil {
		return nil
	}
	key := fmt.Sprintf("%smetrics:%s", r.prefix, userID)
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *riskReadRepository) GetMetrics(ctx context.Context, userID string) (*domain.RiskMetrics, error) {
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

func (r *riskReadRepository) DeleteMetrics(ctx context.Context, userID string) error {
	key := fmt.Sprintf("%smetrics:%s", r.prefix, userID)
	return r.client.Del(ctx, key).Err()
}

func (r *riskReadRepository) SaveCircuitBreaker(ctx context.Context, userID string, cb *domain.CircuitBreaker) error {
	if cb == nil {
		return nil
	}
	key := fmt.Sprintf("%scb:%s", r.prefix, userID)
	data, err := json.Marshal(cb)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *riskReadRepository) GetCircuitBreaker(ctx context.Context, userID string) (*domain.CircuitBreaker, error) {
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

func (r *riskReadRepository) DeleteCircuitBreaker(ctx context.Context, userID string) error {
	key := fmt.Sprintf("%scb:%s", r.prefix, userID)
	return r.client.Del(ctx, key).Err()
}
