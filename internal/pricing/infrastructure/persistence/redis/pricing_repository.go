package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
)

type PricingRedisRepository struct {
	client       redis.UniversalClient
	pricePrefix  string
	resultPrefix string
	ttl          time.Duration
}

func NewPricingRedisRepository(client redis.UniversalClient) *PricingRedisRepository {
	return &PricingRedisRepository{
		client:       client,
		pricePrefix:  "price:",
		resultPrefix: "pricing_result:",
		ttl:          15 * time.Minute,
	}
}

func (r *PricingRedisRepository) SavePrice(ctx context.Context, price *domain.Price) error {
	if price == nil {
		return nil
	}
	data, err := json.Marshal(price)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.priceKey(price.Symbol), data, r.ttl).Err()
}

func (r *PricingRedisRepository) GetLatestPrice(ctx context.Context, symbol string) (*domain.Price, error) {
	if symbol == "" {
		return nil, nil
	}
	data, err := r.client.Get(ctx, r.priceKey(symbol)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var price domain.Price
	if err := json.Unmarshal(data, &price); err != nil {
		return nil, err
	}
	return &price, nil
}

func (r *PricingRedisRepository) SavePricingResult(ctx context.Context, result *domain.PricingResult) error {
	if result == nil {
		return nil
	}
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.resultKey(result.Symbol), data, r.ttl).Err()
}

func (r *PricingRedisRepository) GetLatestPricingResult(ctx context.Context, symbol string) (*domain.PricingResult, error) {
	if symbol == "" {
		return nil, nil
	}
	data, err := r.client.Get(ctx, r.resultKey(symbol)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var result domain.PricingResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *PricingRedisRepository) priceKey(symbol string) string {
	return fmt.Sprintf("%s%s", r.pricePrefix, symbol)
}

func (r *PricingRedisRepository) resultKey(symbol string) string {
	return fmt.Sprintf("%s%s", r.resultPrefix, symbol)
}
