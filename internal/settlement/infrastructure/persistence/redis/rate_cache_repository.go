// Package redis 汇率缓存实现
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/wyfcoding/financialtrading/internal/settlement/domain"
)

type FXRateCacheRepository struct {
	client *redis.Client
}

func NewFXRateCacheRepository(client *redis.Client) *FXRateCacheRepository {
	return &FXRateCacheRepository{client: client}
}

func (r *FXRateCacheRepository) GetRate(ctx context.Context, from, to string) (*domain.FXRate, error) {
	key := fmt.Sprintf("fxrate:%s:%s", from, to)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var rate domain.FXRate
	if err := json.Unmarshal(data, &rate); err != nil {
		return nil, err
	}
	return &rate, nil
}

func (r *FXRateCacheRepository) SaveRate(ctx context.Context, rate *domain.FXRate) error {
	key := fmt.Sprintf("fxrate:%s:%s", rate.FromCurrency, rate.ToCurrency)
	data, _ := json.Marshal(rate)
	return r.client.Set(ctx, key, data, 24*time.Hour).Err()
}
