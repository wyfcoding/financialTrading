package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
	"gorm.io/gorm"
)

type marketDataRepository struct {
	db *gorm.DB
}

// NewMarketDataRepository 创建市场数据仓储实例
func NewMarketDataRepository(db *gorm.DB) domain.MarketDataRepository {
	return &marketDataRepository{db: db}
}

// --- Quote ---

func (r *marketDataRepository) SaveQuote(ctx context.Context, quote *domain.Quote) error {
	return r.db.WithContext(ctx).Create(quote).Error
}

func (r *marketDataRepository) GetLatestQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	var quote domain.Quote
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp desc").First(&quote).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &quote, err
}

// --- Kline ---

func (r *marketDataRepository) SaveKline(ctx context.Context, kline *domain.Kline) error {
	return r.db.WithContext(ctx).Save(kline).Error
}

func (r *marketDataRepository) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*domain.Kline, error) {
	var klines []*domain.Kline
	err := r.db.WithContext(ctx).
		Where("symbol = ? AND interval_period = ?", symbol, interval).
		Order("open_time desc").
		Limit(limit).
		Find(&klines).Error
	return klines, err
}

func (r *marketDataRepository) GetLatestKline(ctx context.Context, symbol, interval string) (*domain.Kline, error) {
	var kline domain.Kline
	err := r.db.WithContext(ctx).
		Where("symbol = ? AND interval_period = ?", symbol, interval).
		Order("open_time desc").
		First(&kline).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &kline, err
}

// --- Trade ---

func (r *marketDataRepository) SaveTrade(ctx context.Context, trade *domain.Trade) error {
	return r.db.WithContext(ctx).Create(trade).Error
}

func (r *marketDataRepository) GetTrades(ctx context.Context, symbol string, limit int) ([]*domain.Trade, error) {
	var trades []*domain.Trade
	err := r.db.WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("timestamp desc").
		Limit(limit).
		Find(&trades).Error
	return trades, err
}

// --- OrderBook ---

func (r *marketDataRepository) SaveOrderBook(ctx context.Context, ob *domain.OrderBook) error {
	return r.db.WithContext(ctx).Save(ob).Error
}

func (r *marketDataRepository) GetOrderBook(ctx context.Context, symbol string) (*domain.OrderBook, error) {
	var ob domain.OrderBook
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&ob).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ob, err
}
