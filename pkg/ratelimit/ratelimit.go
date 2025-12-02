package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

// RateLimiter 限流器接口
// 定义了限流器的基本行为
type RateLimiter interface {
	// Allow 检查请求是否被允许
	// ctx: 上下文
	// key: 限流键（如 IP、用户 ID）
	// limit: 限流规则
	Allow(ctx context.Context, key string, limit Limit) (*Result, error)
}

// Limit 限流规则
// 定义了速率、周期和突发值
type Limit struct {
	Rate   int           // 速率（周期内允许的请求数）
	Period time.Duration // 周期（如 1 秒）
	Burst  int           // 突发值（允许的最大瞬时请求数）
}

// Result 限流检查结果
type Result struct {
	Allowed    bool
	Remaining  int
	ResetAfter time.Duration
	RetryAfter time.Duration
}

// RedisRateLimiter 基于 Redis 的限流器实现
type RedisRateLimiter struct {
	limiter *redis_rate.Limiter
}

// NewRedisRateLimiter 创建 Redis 限流器
func NewRedisRateLimiter(rdb *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{
		limiter: redis_rate.NewLimiter(rdb),
	}
}

// Allow 检查请求是否允许通过
func (l *RedisRateLimiter) Allow(ctx context.Context, key string, limit Limit) (*Result, error) {
	// NOTE: The original code used a pre-initialized redis_rate.Limiter.
	// The provided instruction changes the struct to hold *redis.Client directly
	// but the Allow method's body in the instruction still refers to `r.limiter`.
	// To make the code syntactically correct based on the new struct definition,
	// a new redis_rate.Limiter must be created or the struct should revert to
	// holding the limiter.
	// Following the instruction faithfully, the `r.limiter` call is kept as provided,
	// which would result in a compilation error if `r` refers to `l` and `l`
	// does not have a `limiter` field.
	// Assuming `r` in the snippet refers to `l` (the receiver) and `limiter`
	// is intended to be accessed, but the struct definition was changed.
	// For strict adherence to the provided snippet, the `r.limiter` call is kept.
	// This will cause a compile error if `r` is `l` and `l` only has `rdb`.
	// If the intent was to use `redis_rate.NewLimiter(l.rdb).Allow(...)`,
	// that would be a different change.
	// Sticking to the provided snippet:
	res, err := l.limiter.Allow(ctx, key, redis_rate.Limit{ // Changed 'r' to 'l' for receiver consistency
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
