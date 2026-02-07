package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
)

type PositionRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewPositionRedisRepository(client redis.UniversalClient) *PositionRedisRepository {
	return &PositionRedisRepository{
		client: client,
		prefix: "position:",
		ttl:    15 * time.Minute,
	}
}

func (r *PositionRedisRepository) Save(ctx context.Context, position *domain.Position) error {
	if position == nil || position.ID == 0 {
		return nil
	}
	key := r.key(strconv.FormatUint(uint64(position.ID), 10))
	data, err := json.Marshal(position)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *PositionRedisRepository) Get(ctx context.Context, positionID string) (*domain.Position, error) {
	if positionID == "" {
		return nil, nil
	}
	data, err := r.client.Get(ctx, r.key(positionID)).Bytes()
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

func (r *PositionRedisRepository) key(id string) string {
	return fmt.Sprintf("%s%s", r.prefix, id)
}
