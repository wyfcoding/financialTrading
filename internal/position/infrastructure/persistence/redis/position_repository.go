package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
)

type positionRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewPositionRedisRepository(client redis.UniversalClient) domain.PositionRepository {
	return &positionRedisRepository{
		client: client,
		prefix: "position:",
		ttl:    24 * time.Hour,
	}
}

func (r *positionRedisRepository) Save(ctx context.Context, position *domain.Position) error {
	key := r.key(position.UserID, position.Symbol)
	data, err := json.Marshal(position)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *positionRedisRepository) Get(ctx context.Context, userID, symbol string) (*domain.Position, error) {
	key := r.key(userID, symbol)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var pos domain.Position
	if err := json.Unmarshal(data, &pos); err != nil {
		return nil, err
	}
	return &pos, nil
}

func (r *positionRedisRepository) ListByUser(ctx context.Context, userID string) ([]*domain.Position, error) {
	return nil, nil // Redis 不支持按前缀列表查询（除非扫描或维护 Set）
}

func (r *positionRedisRepository) Delete(ctx context.Context, userID, symbol string) error {
	return r.client.Del(ctx, r.key(userID, symbol)).Err()
}

func (r *positionRedisRepository) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Position, int64, error) {
	return nil, 0, nil
}

func (r *positionRedisRepository) GetBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*domain.Position, int64, error) {
	return nil, 0, nil
}

func (r *positionRedisRepository) GetByUserSymbol(ctx context.Context, userID, symbol string) (*domain.Position, error) {
	return r.Get(ctx, userID, symbol)
}

func (r *positionRedisRepository) Update(ctx context.Context, position *domain.Position) error {
	return r.Save(ctx, position)
}

func (r *positionRedisRepository) Close(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	return nil // Redis 不支持物理平仓逻辑
}

func (r *positionRedisRepository) ExecWithBarrier(ctx context.Context, barrier any, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (r *positionRedisRepository) key(userID, symbol string) string {
	return fmt.Sprintf("%s%s:%s", r.prefix, userID, symbol)
}
