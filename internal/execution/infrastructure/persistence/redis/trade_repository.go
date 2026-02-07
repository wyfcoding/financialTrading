package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

type tradeRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewTradeRedisRepository(client redis.UniversalClient) domain.TradeReadRepository {
	return &tradeRedisRepository{
		client: client,
		prefix: "execution:trade:order:",
		ttl:    6 * time.Hour,
	}
}

func (r *tradeRedisRepository) Save(ctx context.Context, trade *domain.Trade) error {
	if trade == nil {
		return nil
	}
	key := fmt.Sprintf("%s%s", r.prefix, trade.OrderID)
	data, err := json.Marshal(trade)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *tradeRedisRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Trade, error) {
	key := fmt.Sprintf("%s%s", r.prefix, orderID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var trade domain.Trade
	if err := json.Unmarshal(data, &trade); err != nil {
		return nil, err
	}
	return &trade, nil
}

func (r *tradeRedisRepository) Delete(ctx context.Context, orderID string) error {
	key := fmt.Sprintf("%s%s", r.prefix, orderID)
	return r.client.Del(ctx, key).Err()
}
