package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

type QuoteRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

// NewQuoteRedisRepository 创建一个新的基于 Redis 的报价读模型仓储。
func NewQuoteRedisRepository(client redis.UniversalClient) *QuoteRedisRepository {
	return &QuoteRedisRepository{
		client: client,
		prefix: "marketdata:quote:",
		ttl:    24 * time.Hour,
	}
}

func (r *QuoteRedisRepository) Save(ctx context.Context, quote *domain.Quote) error {
	if quote == nil {
		return nil
	}
	key := r.prefix + quote.Symbol
	data, err := json.Marshal(quote)
	if err != nil {
		return fmt.Errorf("failed to marshal quote: %w", err)
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *QuoteRedisRepository) GetLatest(ctx context.Context, symbol string) (*domain.Quote, error) {
	key := r.prefix + symbol
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get quote from redis: %w", err)
	}

	var quote domain.Quote
	if err := json.Unmarshal(data, &quote); err != nil {
		return nil, fmt.Errorf("failed to unmarshal quote: %w", err)
	}
	return &quote, nil
}
