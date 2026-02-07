package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
)

type OrderRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewOrderRedisRepository(client redis.UniversalClient) *OrderRedisRepository {
	return &OrderRedisRepository{
		client: client,
		prefix: "order:",
		ttl:    15 * time.Minute,
	}
}

func (r *OrderRedisRepository) Save(ctx context.Context, order *domain.Order) error {
	if order == nil {
		return nil
	}
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}
	return r.client.Set(ctx, r.key(order.OrderID), data, r.ttl).Err()
}

func (r *OrderRedisRepository) Get(ctx context.Context, orderID string) (*domain.Order, error) {
	if orderID == "" {
		return nil, nil
	}
	data, err := r.client.Get(ctx, r.key(orderID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get order from redis: %w", err)
	}
	var order domain.Order
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order: %w", err)
	}
	return &order, nil
}

func (r *OrderRedisRepository) key(orderID string) string {
	return r.prefix + orderID
}
