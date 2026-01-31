package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

type quoteRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

// NewQuoteRepository 创建一个新的基于 Redis 的报价仓储。
func NewQuoteRepository(client redis.UniversalClient) domain.MarketDataRepository {
	return &quoteRepository{
		client: client,
		prefix: "marketdata:quote:",
		ttl:    24 * time.Hour, // 缓存 24 小时
	}
}

func (r *quoteRepository) SaveQuote(ctx context.Context, quote *domain.Quote) error {
	key := r.prefix + quote.Symbol
	data, err := json.Marshal(quote)
	if err != nil {
		return fmt.Errorf("failed to marshal quote: %w", err)
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *quoteRepository) GetLatestQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	key := r.prefix + symbol
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get quote from redis: %w", err)
	}

	var quote domain.Quote
	if err := json.Unmarshal(data, &quote); err != nil {
		return nil, fmt.Errorf("failed to unmarshal quote: %w", err)
	}
	return &quote, nil
}

// 以下方法在 RedisQuoteRepository 中暂不实现，或仅作为占位符。
// 在复合仓储中会由 MySQL 实现处理。

func (r *quoteRepository) SaveKline(ctx context.Context, kline *domain.Kline) error { return nil }
func (r *quoteRepository) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*domain.Kline, error) {
	return nil, nil
}
func (r *quoteRepository) GetLatestKline(ctx context.Context, symbol, interval string) (*domain.Kline, error) {
	return nil, nil
}
func (r *quoteRepository) SaveTrade(ctx context.Context, trade *domain.Trade) error { return nil }
func (r *quoteRepository) GetTrades(ctx context.Context, symbol string, limit int) ([]*domain.Trade, error) {
	return nil, nil
}
func (r *quoteRepository) SaveOrderBook(ctx context.Context, ob *domain.OrderBook) error { return nil }
func (r *quoteRepository) GetOrderBook(ctx context.Context, symbol string) (*domain.OrderBook, error) {
	return nil, nil
}
