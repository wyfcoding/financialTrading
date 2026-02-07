package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/connectivity/domain"
)

type quoteRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewQuoteRedisRepository(client redis.UniversalClient) domain.QuoteRepository {
	return &quoteRedisRepository{
		client: client,
		prefix: "connectivity:quote:",
		ttl:    6 * time.Hour,
	}
}

func (r *quoteRedisRepository) Save(ctx context.Context, quote *domain.Quote) error {
	if quote == nil {
		return nil
	}
	key := r.prefix + quote.Symbol
	data, err := json.Marshal(quote)
	if err != nil {
		return fmt.Errorf("marshal quote failed: %w", err)
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *quoteRedisRepository) Get(ctx context.Context, symbol string) (*domain.Quote, error) {
	key := r.prefix + symbol
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var quote domain.Quote
	if err := json.Unmarshal(data, &quote); err != nil {
		return nil, err
	}
	return &quote, nil
}

func (r *quoteRedisRepository) Delete(ctx context.Context, symbol string) error {
	key := r.prefix + symbol
	return r.client.Del(ctx, key).Err()
}
