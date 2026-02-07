package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
)

type MetricRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
	maxLen int64
}

func NewMetricRedisRepository(client redis.UniversalClient) *MetricRedisRepository {
	return &MetricRedisRepository{
		client: client,
		prefix: "monitoring:metric:",
		ttl:    10 * time.Minute,
		maxLen: 2000,
	}
}

func (r *MetricRedisRepository) Save(ctx context.Context, m *domain.Metric) error {
	if m == nil {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}
	key := r.prefix + m.Name
	pipe := r.client.TxPipeline()
	pipe.LPush(ctx, key, data)
	pipe.LTrim(ctx, key, 0, r.maxLen-1)
	pipe.Expire(ctx, key, r.ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *MetricRedisRepository) ListRecent(ctx context.Context, name string, limit int) ([]*domain.Metric, error) {
	if limit <= 0 {
		limit = int(r.maxLen)
	}
	key := r.prefix + name
	values, err := r.client.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list metrics from redis: %w", err)
	}
	res := make([]*domain.Metric, 0, len(values))
	for _, val := range values {
		var m domain.Metric
		if err := json.Unmarshal([]byte(val), &m); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metric: %w", err)
		}
		res = append(res, &m)
	}
	return res, nil
}

type SystemHealthRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
	maxLen int64
}

func NewSystemHealthRedisRepository(client redis.UniversalClient) *SystemHealthRedisRepository {
	return &SystemHealthRedisRepository{
		client: client,
		prefix: "monitoring:health:",
		ttl:    5 * time.Minute,
		maxLen: 200,
	}
}

func (r *SystemHealthRedisRepository) Save(ctx context.Context, h *domain.SystemHealth) error {
	if h == nil {
		return nil
	}
	data, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("failed to marshal system health: %w", err)
	}
	key := r.prefix + h.ServiceName
	pipe := r.client.TxPipeline()
	pipe.LPush(ctx, key, data)
	pipe.LTrim(ctx, key, 0, r.maxLen-1)
	pipe.Expire(ctx, key, r.ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *SystemHealthRedisRepository) ListLatest(ctx context.Context, serviceName string, limit int) ([]*domain.SystemHealth, error) {
	if limit <= 0 {
		limit = int(r.maxLen)
	}
	key := r.prefix + serviceName
	values, err := r.client.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list health from redis: %w", err)
	}
	res := make([]*domain.SystemHealth, 0, len(values))
	for _, val := range values {
		var h domain.SystemHealth
		if err := json.Unmarshal([]byte(val), &h); err != nil {
			return nil, fmt.Errorf("failed to unmarshal system health: %w", err)
		}
		res = append(res, &h)
	}
	return res, nil
}

type AlertRedisRepository struct {
	client redis.UniversalClient
	key    string
	ttl    time.Duration
	maxLen int64
}

func NewAlertRedisRepository(client redis.UniversalClient) *AlertRedisRepository {
	return &AlertRedisRepository{
		client: client,
		key:    "monitoring:alert",
		ttl:    10 * time.Minute,
		maxLen: 200,
	}
}

func (r *AlertRedisRepository) Save(ctx context.Context, a *domain.Alert) error {
	if a == nil {
		return nil
	}
	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}
	pipe := r.client.TxPipeline()
	pipe.LPush(ctx, r.key, data)
	pipe.LTrim(ctx, r.key, 0, r.maxLen-1)
	pipe.Expire(ctx, r.key, r.ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *AlertRedisRepository) ListLatest(ctx context.Context, limit int) ([]*domain.Alert, error) {
	if limit <= 0 {
		limit = int(r.maxLen)
	}
	values, err := r.client.LRange(ctx, r.key, 0, int64(limit-1)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list alerts from redis: %w", err)
	}
	res := make([]*domain.Alert, 0, len(values))
	for _, val := range values {
		var a domain.Alert
		if err := json.Unmarshal([]byte(val), &a); err != nil {
			return nil, fmt.Errorf("failed to unmarshal alert: %w", err)
		}
		res = append(res, &a)
	}
	return res, nil
}
