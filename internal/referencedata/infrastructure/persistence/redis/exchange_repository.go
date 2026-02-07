package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

type ExchangeRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

// NewExchangeRedisRepository 创建基于 Redis 的交易所读模型仓储
func NewExchangeRedisRepository(client redis.UniversalClient) *ExchangeRedisRepository {
	return &ExchangeRedisRepository{
		client: client,
		prefix: "referencedata:exchange:",
		ttl:    24 * time.Hour,
	}
}

func (r *ExchangeRedisRepository) Save(ctx context.Context, exchange *domain.Exchange) error {
	if exchange == nil {
		return nil
	}
	data, err := json.Marshal(exchange)
	if err != nil {
		return fmt.Errorf("failed to marshal exchange: %w", err)
	}
	if exchange.ID != "" {
		if err := r.client.Set(ctx, r.prefix+"id:"+exchange.ID, data, r.ttl).Err(); err != nil {
			return err
		}
	}
	if exchange.Name != "" {
		if err := r.client.Set(ctx, r.prefix+"name:"+exchange.Name, data, r.ttl).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (r *ExchangeRedisRepository) Get(ctx context.Context, id string) (*domain.Exchange, error) {
	if id == "" {
		return nil, nil
	}
	data, err := r.client.Get(ctx, r.prefix+"id:"+id).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get exchange from redis: %w", err)
	}
	var exchange domain.Exchange
	if err := json.Unmarshal(data, &exchange); err != nil {
		return nil, fmt.Errorf("failed to unmarshal exchange: %w", err)
	}
	return &exchange, nil
}

func (r *ExchangeRedisRepository) GetByName(ctx context.Context, name string) (*domain.Exchange, error) {
	if name == "" {
		return nil, nil
	}
	data, err := r.client.Get(ctx, r.prefix+"name:"+name).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get exchange by name from redis: %w", err)
	}
	var exchange domain.Exchange
	if err := json.Unmarshal(data, &exchange); err != nil {
		return nil, fmt.Errorf("failed to unmarshal exchange: %w", err)
	}
	return &exchange, nil
}
