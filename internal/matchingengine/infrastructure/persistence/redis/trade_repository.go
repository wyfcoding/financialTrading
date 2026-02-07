package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
)

type TradeRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
	maxLen int64
}

func NewTradeRedisRepository(client redis.UniversalClient) *TradeRedisRepository {
	return &TradeRedisRepository{
		client: client,
		prefix: "matching:trade:",
		ttl:    24 * time.Hour,
		maxLen: 1000,
	}
}

func (r *TradeRedisRepository) Save(ctx context.Context, trade *domain.Trade) error {
	if trade == nil {
		return nil
	}
	key := r.prefix + trade.Symbol
	data, err := json.Marshal(trade)
	if err != nil {
		return fmt.Errorf("failed to marshal trade: %w", err)
	}
	pipe := r.client.TxPipeline()
	pipe.LPush(ctx, key, data)
	pipe.LTrim(ctx, key, 0, r.maxLen-1)
	pipe.Expire(ctx, key, r.ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *TradeRedisRepository) List(ctx context.Context, symbol string, limit int) ([]*domain.Trade, error) {
	key := r.prefix + symbol
	if limit <= 0 {
		limit = int(r.maxLen)
	}
	values, err := r.client.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list trades from redis: %w", err)
	}
	trades := make([]*domain.Trade, 0, len(values))
	for _, val := range values {
		var trade domain.Trade
		if err := json.Unmarshal([]byte(val), &trade); err != nil {
			return nil, fmt.Errorf("failed to unmarshal trade: %w", err)
		}
		trades = append(trades, &trade)
	}
	return trades, nil
}
