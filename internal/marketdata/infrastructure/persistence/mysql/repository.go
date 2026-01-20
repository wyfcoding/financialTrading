package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
	"gorm.io/gorm"
)

type quoteRepository struct {
	db *gorm.DB
}

func NewQuoteRepository(db *gorm.DB) domain.QuoteRepository {
	return &quoteRepository{db: db}
}

func (r *quoteRepository) Save(ctx context.Context, quote *domain.Quote) error {
	var po QuotePO
	po.FromDomain(quote)
	return r.db.WithContext(ctx).Create(&po).Error
}

func (r *quoteRepository) GetLatest(ctx context.Context, symbol string) (*domain.Quote, error) {
	var po QuotePO
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp desc").First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return po.ToDomain(), nil
}

type klineRepository struct {
	db *gorm.DB
}

func NewKlineRepository(db *gorm.DB) domain.KlineRepository {
	return &klineRepository{db: db}
}

func (r *klineRepository) Save(ctx context.Context, kline *domain.Kline) error {
	var po KlinePO
	po.FromDomain(kline)
	return r.db.WithContext(ctx).Create(&po).Error
}

func (r *klineRepository) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*domain.Kline, error) {
	var pos []*KlinePO
	err := r.db.WithContext(ctx).
		Where("symbol = ?", symbol). // Corrected the partial line here
		Where("interval_str = ?", interval).
		Order("open_time desc").
		Limit(limit).
		Find(&pos).Error
	if err != nil {
		return nil, err
	}

	res := make([]*domain.Kline, len(pos))
	for i, po := range pos {
		res[i] = po.ToDomain()
	}
	return res, nil
}

type tradeRepository struct {
	db *gorm.DB
}

func NewTradeRepository(db *gorm.DB) domain.TradeRepository {
	return &tradeRepository{db: db}
}

func (r *tradeRepository) Save(ctx context.Context, trade *domain.Trade) error {
	var po TradePO
	po.FromDomain(trade)
	return r.db.WithContext(ctx).Create(&po).Error
}

func (r *tradeRepository) GetTrades(ctx context.Context, symbol string, limit int) ([]*domain.Trade, error) {
	var pos []*TradePO
	err := r.db.WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("time desc").
		Limit(limit).
		Find(&pos).Error
	if err != nil {
		return nil, err
	}

	res := make([]*domain.Trade, len(pos))
	for i, po := range pos {
		res[i] = po.ToDomain()
	}
	return res, nil
}

type orderBookRepository struct {
	db *gorm.DB
}

func NewOrderBookRepository(db *gorm.DB) domain.OrderBookRepository {
	return &orderBookRepository{db: db}
}

func (r *orderBookRepository) Save(ctx context.Context, ob *domain.OrderBook) error {
	// Simple implementation: overwrite snapshot or ignore if we don't have PO for it yet.
	// For now, return nil to satisfy interface if we don't want to add complexity.
	return nil
}

func (r *orderBookRepository) Get(ctx context.Context, symbol string) (*domain.OrderBook, error) {
	return nil, nil
}
