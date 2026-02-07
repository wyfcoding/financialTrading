package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

type SymbolRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

// NewSymbolRedisRepository 创建基于 Redis 的交易对读模型仓储
func NewSymbolRedisRepository(client redis.UniversalClient) *SymbolRedisRepository {
	return &SymbolRedisRepository{
		client: client,
		prefix: "referencedata:symbol:",
		ttl:    24 * time.Hour,
	}
}

func (r *SymbolRedisRepository) Save(ctx context.Context, symbol *domain.Symbol) error {
	if symbol == nil {
		return nil
	}
	data, err := json.Marshal(symbol)
	if err != nil {
		return fmt.Errorf("failed to marshal symbol: %w", err)
	}
	if symbol.ID != "" {
		if err := r.client.Set(ctx, r.prefix+"id:"+symbol.ID, data, r.ttl).Err(); err != nil {
			return err
		}
	}
	if symbol.SymbolCode != "" {
		if err := r.client.Set(ctx, r.prefix+"code:"+symbol.SymbolCode, data, r.ttl).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (r *SymbolRedisRepository) Get(ctx context.Context, id string) (*domain.Symbol, error) {
	if id == "" {
		return nil, nil
	}
	data, err := r.client.Get(ctx, r.prefix+"id:"+id).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get symbol from redis: %w", err)
	}
	var symbol domain.Symbol
	if err := json.Unmarshal(data, &symbol); err != nil {
		return nil, fmt.Errorf("failed to unmarshal symbol: %w", err)
	}
	return &symbol, nil
}

func (r *SymbolRedisRepository) GetByCode(ctx context.Context, code string) (*domain.Symbol, error) {
	if code == "" {
		return nil, nil
	}
	data, err := r.client.Get(ctx, r.prefix+"code:"+code).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get symbol by code from redis: %w", err)
	}
	var symbol domain.Symbol
	if err := json.Unmarshal(data, &symbol); err != nil {
		return nil, fmt.Errorf("failed to unmarshal symbol: %w", err)
	}
	return &symbol, nil
}
