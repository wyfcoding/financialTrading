package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
)

type accountRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewAccountRedisRepository(client redis.UniversalClient) domain.AccountReadRepository {
	return &accountRedisRepository{
		client: client,
		prefix: "account:",
		ttl:    24 * time.Hour,
	}
}

func (r *accountRedisRepository) Save(ctx context.Context, account *domain.Account) error {
	if account == nil {
		return nil
	}
	key := r.key(account.AccountID)
	data, err := json.Marshal(account)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *accountRedisRepository) Get(ctx context.Context, accountID string) (*domain.Account, error) {
	key := r.key(accountID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var account domain.Account
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *accountRedisRepository) Delete(ctx context.Context, accountID string) error {
	return r.client.Del(ctx, r.key(accountID)).Err()
}

func (r *accountRedisRepository) key(id string) string {
	return r.prefix + id
}
