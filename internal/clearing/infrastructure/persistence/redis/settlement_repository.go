package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
)

type settlementRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewSettlementRedisRepository(client redis.UniversalClient) domain.SettlementReadRepository {
	return &settlementRedisRepository{
		client: client,
		prefix: "clearing:settlement:",
		ttl:    6 * time.Hour,
	}
}

func (r *settlementRedisRepository) Save(ctx context.Context, settlement *domain.Settlement) error {
	if settlement == nil {
		return nil
	}
	key := fmt.Sprintf("%s%s", r.prefix, settlement.SettlementID)
	data, err := json.Marshal(settlement)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *settlementRedisRepository) Get(ctx context.Context, settlementID string) (*domain.Settlement, error) {
	key := fmt.Sprintf("%s%s", r.prefix, settlementID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var settlement domain.Settlement
	if err := json.Unmarshal(data, &settlement); err != nil {
		return nil, err
	}
	return &settlement, nil
}

func (r *settlementRedisRepository) Delete(ctx context.Context, settlementID string) error {
	key := fmt.Sprintf("%s%s", r.prefix, settlementID)
	return r.client.Del(ctx, key).Err()
}
