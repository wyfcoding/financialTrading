package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

type InstrumentRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

// NewInstrumentRedisRepository 创建基于 Redis 的合约读模型仓储
func NewInstrumentRedisRepository(client redis.UniversalClient) *InstrumentRedisRepository {
	return &InstrumentRedisRepository{
		client: client,
		prefix: "referencedata:instrument:",
		ttl:    24 * time.Hour,
	}
}

func (r *InstrumentRedisRepository) Save(ctx context.Context, instrument *domain.Instrument) error {
	if instrument == nil {
		return nil
	}
	data, err := json.Marshal(instrument)
	if err != nil {
		return fmt.Errorf("failed to marshal instrument: %w", err)
	}
	if instrument.Symbol != "" {
		if err := r.client.Set(ctx, r.prefix+"symbol:"+instrument.Symbol, data, r.ttl).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (r *InstrumentRedisRepository) Get(ctx context.Context, symbol string) (*domain.Instrument, error) {
	if symbol == "" {
		return nil, nil
	}
	data, err := r.client.Get(ctx, r.prefix+"symbol:"+symbol).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get instrument from redis: %w", err)
	}
	var instrument domain.Instrument
	if err := json.Unmarshal(data, &instrument); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instrument: %w", err)
	}
	return &instrument, nil
}
