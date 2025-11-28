package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	// Allow checks if the request is allowed for the given key and limit
	Allow(ctx context.Context, key string, limit Limit) (*Result, error)
}

// Limit defines the rate limit rule
type Limit struct {
	Rate   int
	Period time.Duration
	Burst  int
}

// Result represents the result of a rate limit check
type Result struct {
	Allowed    bool
	Remaining  int
	ResetAfter time.Duration
	RetryAfter time.Duration
}

// RedisRateLimiter implements RateLimiter using Redis
type RedisRateLimiter struct {
	limiter *redis_rate.Limiter
}

// NewRedisRateLimiter creates a new RedisRateLimiter
func NewRedisRateLimiter(rdb *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{
		limiter: redis_rate.NewLimiter(rdb),
	}
}

// Allow checks if the request is allowed
func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit Limit) (*Result, error) {
	res, err := r.limiter.Allow(ctx, key, redis_rate.Limit{
		Rate:   limit.Rate,
		Period: limit.Period,
		Burst:  limit.Burst,
	})
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}

	return &Result{
		Allowed:    res.Allowed > 0,
		Remaining:  res.Remaining,
		ResetAfter: res.ResetAfter,
		RetryAfter: res.RetryAfter,
	}, nil
}
