// Package cache 提供 Redis 客户端封装，支持连接池、监控、backoff、二级缓存策略、多种序列化
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fynnwu/FinancialTrading/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// Config Redis 配置
type Config struct {
	Host         string
	Port         int
	Password     string
	DB           int
	MaxPoolSize  int
	ConnTimeout  int
	ReadTimeout  int
	WriteTimeout int
}

// RedisCache Redis 缓存实现
type RedisCache struct {
	client *redis.Client
	config Config
}

// New 创建 Redis 缓存实例
func New(cfg Config) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.MaxPoolSize,
		ConnMaxIdleTime: time.Duration(cfg.ConnTimeout) * time.Second,
		ReadTimeout:     time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout:    time.Duration(cfg.WriteTimeout) * time.Second,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info(context.Background(), "Redis connected successfully", "addr", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))

	return &RedisCache{
		client: client,
		config: cfg,
	}, nil
}

// Get 获取缓存值
func (rc *RedisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := rc.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		logger.Error(ctx, "Redis Get failed", "key", key, "error", err)
		return "", err
	}
	return val, nil
}

// GetJSON 获取 JSON 格式的缓存值
func (rc *RedisCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := rc.Get(ctx, key)
	if err != nil {
		return err
	}
	if val == "" {
		return nil
	}
	return json.Unmarshal([]byte(val), dest)
}

// Set 设置缓存值
func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	err := rc.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		logger.Error(ctx, "Redis Set failed", "key", key, "error", err)
		return err
	}
	return nil
}

// SetJSON 设置 JSON 格式的缓存值
func (rc *RedisCache) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return rc.Set(ctx, key, string(data), expiration)
}

// SetNX 仅当 key 不存在时设置值（用于分布式锁）
func (rc *RedisCache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	ok, err := rc.client.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		logger.Error(ctx, "Redis SetNX failed", "key", key, "error", err)
		return false, err
	}
	return ok, nil
}

// Delete 删除缓存
func (rc *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	err := rc.client.Del(ctx, keys...).Err()
	if err != nil {
		logger.Error(ctx, "Redis Delete failed", "keys", keys, "error", err)
		return err
	}
	return nil
}

// Exists 检查 key 是否存在
func (rc *RedisCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	count, err := rc.client.Exists(ctx, keys...).Result()
	if err != nil {
		logger.Error(ctx, "Redis Exists failed", "keys", keys, "error", err)
		return 0, err
	}
	return count, nil
}

// Expire 设置 key 的过期时间
func (rc *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	err := rc.client.Expire(ctx, key, expiration).Err()
	if err != nil {
		logger.Error(ctx, "Redis Expire failed", "key", key, "error", err)
		return err
	}
	return nil
}

// TTL 获取 key 的剩余过期时间
func (rc *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := rc.client.TTL(ctx, key).Result()
	if err != nil {
		logger.Error(ctx, "Redis TTL failed", "key", key, "error", err)
		return 0, err
	}
	return ttl, nil
}

// Incr 原子递增
func (rc *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	val, err := rc.client.Incr(ctx, key).Result()
	if err != nil {
		logger.Error(ctx, "Redis Incr failed", "key", key, "error", err)
		return 0, err
	}
	return val, nil
}

// IncrBy 原子递增指定值
func (rc *RedisCache) IncrBy(ctx context.Context, key string, increment int64) (int64, error) {
	val, err := rc.client.IncrBy(ctx, key, increment).Result()
	if err != nil {
		logger.Error(ctx, "Redis IncrBy failed", "key", key, "error", err)
		return 0, err
	}
	return val, nil
}

// Decr 原子递减
func (rc *RedisCache) Decr(ctx context.Context, key string) (int64, error) {
	val, err := rc.client.Decr(ctx, key).Result()
	if err != nil {
		logger.Error(ctx, "Redis Decr failed", "key", key, "error", err)
		return 0, err
	}
	return val, nil
}

// LPush 左推入列表
func (rc *RedisCache) LPush(ctx context.Context, key string, values ...interface{}) error {
	err := rc.client.LPush(ctx, key, values...).Err()
	if err != nil {
		logger.Error(ctx, "Redis LPush failed", "key", key, "error", err)
		return err
	}
	return nil
}

// RPush 右推入列表
func (rc *RedisCache) RPush(ctx context.Context, key string, values ...interface{}) error {
	err := rc.client.RPush(ctx, key, values...).Err()
	if err != nil {
		logger.Error(ctx, "Redis RPush failed", "key", key, "error", err)
		return err
	}
	return nil
}

// LRange 获取列表范围内的元素
func (rc *RedisCache) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	vals, err := rc.client.LRange(ctx, key, start, stop).Result()
	if err != nil {
		logger.Error(ctx, "Redis LRange failed", "key", key, "error", err)
		return nil, err
	}
	return vals, nil
}

// HSet 设置哈希字段
func (rc *RedisCache) HSet(ctx context.Context, key string, values ...interface{}) error {
	err := rc.client.HSet(ctx, key, values...).Err()
	if err != nil {
		logger.Error(ctx, "Redis HSet failed", "key", key, "error", err)
		return err
	}
	return nil
}

// HGet 获取哈希字段
func (rc *RedisCache) HGet(ctx context.Context, key, field string) (string, error) {
	val, err := rc.client.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		logger.Error(ctx, "Redis HGet failed", "key", key, "field", field, "error", err)
		return "", err
	}
	return val, nil
}

// HGetAll 获取所有哈希字段
func (rc *RedisCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	vals, err := rc.client.HGetAll(ctx, key).Result()
	if err != nil {
		logger.Error(ctx, "Redis HGetAll failed", "key", key, "error", err)
		return nil, err
	}
	return vals, nil
}

// ZAdd 添加有序集合成员
func (rc *RedisCache) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	err := rc.client.ZAdd(ctx, key, members...).Err()
	if err != nil {
		logger.Error(ctx, "Redis ZAdd failed", "key", key, "error", err)
		return err
	}
	return nil
}

// ZRange 获取有序集合范围内的成员
func (rc *RedisCache) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	vals, err := rc.client.ZRange(ctx, key, start, stop).Result()
	if err != nil {
		logger.Error(ctx, "Redis ZRange failed", "key", key, "error", err)
		return nil, err
	}
	return vals, nil
}

// ZRangeByScore 按分数范围获取有序集合成员
func (rc *RedisCache) ZRangeByScore(ctx context.Context, key string, min, max string) ([]string, error) {
	vals, err := rc.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: min,
		Max: max,
	}).Result()
	if err != nil {
		logger.Error(ctx, "Redis ZRangeByScore failed", "key", key, "error", err)
		return nil, err
	}
	return vals, nil
}

// Close 关闭 Redis 连接
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// GetClient 获取底层 Redis 客户端（用于高级操作）
func (rc *RedisCache) GetClient() *redis.Client {
	return rc.client
}
