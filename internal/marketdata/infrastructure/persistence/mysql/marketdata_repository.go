package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type marketDataRepository struct {
	db *gorm.DB
}

// NewMarketDataRepository 创建市场数据仓储实例
func NewMarketDataRepository(db *gorm.DB) domain.MarketDataRepository {
	return &marketDataRepository{db: db}
}

// --- tx helpers ---

func (r *marketDataRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *marketDataRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *marketDataRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *marketDataRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

// --- Quote ---

func (r *marketDataRepository) SaveQuote(ctx context.Context, quote *domain.Quote) error {
	model := toQuoteModel(quote)
	return r.getDB(ctx).WithContext(ctx).Create(model).Error
}

func (r *marketDataRepository) GetLatestQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	var model QuoteModel
	err := r.getDB(ctx).WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("timestamp desc").
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toQuote(&model), err
}

// --- Kline ---

func (r *marketDataRepository) SaveKline(ctx context.Context, kline *domain.Kline) error {
	model := toKlineModel(kline)
	db := r.getDB(ctx).WithContext(ctx)

	var existing KlineModel
	err := db.Where("symbol = ? AND interval_period = ? AND open_time = ?", model.Symbol, model.Interval, model.OpenTime).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(model).Error
	}
	if err != nil {
		return err
	}

	return db.Model(&KlineModel{}).
		Where("id = ?", existing.ID).
		Updates(map[string]any{
			"close_time": model.CloseTime,
			"open":       model.Open,
			"high":       model.High,
			"low":        model.Low,
			"close":      model.Close,
			"volume":     model.Volume,
		}).Error
}

func (r *marketDataRepository) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*domain.Kline, error) {
	var models []*KlineModel
	err := r.getDB(ctx).WithContext(ctx).
		Where("symbol = ? AND interval_period = ?", symbol, interval).
		Order("open_time desc").
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	klines := make([]*domain.Kline, len(models))
	for i, m := range models {
		klines[i] = toKline(m)
	}
	return klines, nil
}

func (r *marketDataRepository) GetLatestKline(ctx context.Context, symbol, interval string) (*domain.Kline, error) {
	var model KlineModel
	err := r.getDB(ctx).WithContext(ctx).
		Where("symbol = ? AND interval_period = ?", symbol, interval).
		Order("open_time desc").
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toKline(&model), err
}

// --- Trade ---

func (r *marketDataRepository) SaveTrade(ctx context.Context, trade *domain.Trade) error {
	model := toTradeModel(trade)
	return r.getDB(ctx).WithContext(ctx).Create(model).Error
}

func (r *marketDataRepository) GetTrades(ctx context.Context, symbol string, limit int) ([]*domain.Trade, error) {
	var models []*TradeModel
	err := r.getDB(ctx).WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("timestamp desc").
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	trades := make([]*domain.Trade, len(models))
	for i, m := range models {
		trades[i] = toTrade(m)
	}
	return trades, nil
}

// --- OrderBook ---

func (r *marketDataRepository) SaveOrderBook(ctx context.Context, ob *domain.OrderBook) error {
	model, err := toOrderBookModel(ob)
	if err != nil {
		return err
	}
	if model == nil {
		return nil
	}

	db := r.getDB(ctx).WithContext(ctx)
	var existing OrderBookModel
	err = db.Where("symbol = ?", model.Symbol).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(model).Error
	}
	if err != nil {
		return err
	}

	return db.Model(&OrderBookModel{}).
		Where("id = ?", existing.ID).
		Updates(map[string]any{
			"bids":      model.BidsJSON,
			"asks":      model.AsksJSON,
			"timestamp": model.Timestamp,
		}).Error
}

func (r *marketDataRepository) GetOrderBook(ctx context.Context, symbol string) (*domain.OrderBook, error) {
	var model OrderBookModel
	err := r.getDB(ctx).WithContext(ctx).
		Where("symbol = ?", symbol).
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toOrderBook(&model)
}

func (r *marketDataRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
