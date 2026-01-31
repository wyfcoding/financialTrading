package persistence

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

type compositeMarketDataRepository struct {
	mysql domain.MarketDataRepository
	redis domain.MarketDataRepository
}

// NewCompositeMarketDataRepository 创建一个组合仓储，支持 MySQL 持久化和 Redis 缓存。
func NewCompositeMarketDataRepository(mysql, redis domain.MarketDataRepository) domain.MarketDataRepository {
	return &compositeMarketDataRepository{
		mysql: mysql,
		redis: redis,
	}
}

func (r *compositeMarketDataRepository) SaveQuote(ctx context.Context, quote *domain.Quote) error {
	// 双写：先写 MySQL (持久化)，再写 Redis (缓存)
	if err := r.mysql.SaveQuote(ctx, quote); err != nil {
		return err
	}
	return r.redis.SaveQuote(ctx, quote)
}

func (r *compositeMarketDataRepository) GetLatestQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	// 先读 Redis
	quote, err := r.redis.GetLatestQuote(ctx, symbol)
	if err == nil && quote != nil {
		return quote, nil
	}
	// Redis 不存在则读 MySQL
	return r.mysql.GetLatestQuote(ctx, symbol)
}

func (r *compositeMarketDataRepository) SaveKline(ctx context.Context, kline *domain.Kline) error {
	return r.mysql.SaveKline(ctx, kline)
}

func (r *compositeMarketDataRepository) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*domain.Kline, error) {
	return r.mysql.GetKlines(ctx, symbol, interval, limit)
}

func (r *compositeMarketDataRepository) GetLatestKline(ctx context.Context, symbol, interval string) (*domain.Kline, error) {
	return r.mysql.GetLatestKline(ctx, symbol, interval)
}

func (r *compositeMarketDataRepository) SaveTrade(ctx context.Context, trade *domain.Trade) error {
	return r.mysql.SaveTrade(ctx, trade)
}

func (r *compositeMarketDataRepository) GetTrades(ctx context.Context, symbol string, limit int) ([]*domain.Trade, error) {
	return r.mysql.GetTrades(ctx, symbol, limit)
}

func (r *compositeMarketDataRepository) SaveOrderBook(ctx context.Context, ob *domain.OrderBook) error {
	return r.mysql.SaveOrderBook(ctx, ob)
}

func (r *compositeMarketDataRepository) GetOrderBook(ctx context.Context, symbol string) (*domain.OrderBook, error) {
	return r.mysql.GetOrderBook(ctx, symbol)
}
