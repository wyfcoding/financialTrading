package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/auth/domain"
)

type sessionRedisRepository struct {
	client redis.UniversalClient
	prefix string
}

func NewSessionRedisRepository(client redis.UniversalClient) domain.SessionRepository {
	return &sessionRedisRepository{
		client: client,
		prefix: "auth:session:",
	}
}

func (r *sessionRedisRepository) Save(ctx context.Context, session *domain.AuthSession) error {
	key := fmt.Sprintf("%s%s", r.prefix, session.Token)
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session already expired")
	}

	return r.client.Set(ctx, key, data, ttl).Err()
}

func (r *sessionRedisRepository) Get(ctx context.Context, token string) (*domain.AuthSession, error) {
	key := fmt.Sprintf("%s%s", r.prefix, token)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var session domain.AuthSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *sessionRedisRepository) Delete(ctx context.Context, token string) error {
	key := fmt.Sprintf("%s%s", r.prefix, token)
	return r.client.Del(ctx, key).Err()
}

func (r *sessionRedisRepository) DeleteByUserID(ctx context.Context, userID uint) error {
	// 实际上需要一个 userID -> token 的映射或者扫描键
	// 在工业级实现中，通常会维护一个 set auth:user_sessions:{userID}
	pattern := fmt.Sprintf("%s*", r.prefix)
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		data, err := r.client.Get(ctx, key).Bytes()
		if err == nil {
			var session domain.AuthSession
			if err := json.Unmarshal(data, &session); err == nil && session.UserID == userID {
				_ = r.client.Del(ctx, key)
			}
		}
	}
	return iter.Err()
}
