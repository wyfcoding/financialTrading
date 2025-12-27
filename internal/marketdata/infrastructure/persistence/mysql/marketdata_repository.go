// Package mysql 提供了市场数据仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- Quote ---

type QuoteModel struct {
	gorm.Model
	Symbol    string `gorm:"column:symbol;type:varchar(20);index;not null"`
	BidPrice  string `gorm:"column:bid_price;type:decimal(32,18);not null"`
	AskPrice  string `gorm:"column:ask_price;type:decimal(32,18);not null"`
	BidSize   string `gorm:"column:bid_size;type:decimal(32,18);not null"`
	AskSize   string `gorm:"column:ask_size;type:decimal(32,18);not null"`
	LastPrice string `gorm:"column:last_price;type:decimal(32,18);not null"`
	LastSize  string `gorm:"column:last_size;type:decimal(32,18);not null"`
	Timestamp int64  `gorm:"column:timestamp;type:bigint;not null"`
	Source    string `gorm:"column:source;type:varchar(50)"`
}

func (QuoteModel) TableName() string { return "quotes" }

type quoteRepositoryImpl struct {
	db *gorm.DB
}

func NewQuoteRepository(db *gorm.DB) domain.QuoteRepository {
	return &quoteRepositoryImpl{db: db}
}

func (r *quoteRepositoryImpl) Save(ctx context.Context, q *domain.Quote) error {
	m := &QuoteModel{
		Model:     q.Model,
		Symbol:    q.Symbol,
		BidPrice:  q.BidPrice.String(),
		AskPrice:  q.AskPrice.String(),
		BidSize:   q.BidSize.String(),
		AskSize:   q.AskSize.String(),
		LastPrice: q.LastPrice.String(),
		LastSize:  q.LastSize.String(),
		Timestamp: q.Timestamp,
		Source:    q.Source,
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(m).Error
	if err == nil {
		q.Model = m.Model
	}
	return err
}

func (r *quoteRepositoryImpl) GetLatest(ctx context.Context, symbol string) (*domain.Quote, error) {
	var m QuoteModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp desc").First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

func (r *quoteRepositoryImpl) GetHistory(ctx context.Context, symbol string, startTime, endTime int64) ([]*domain.Quote, error) {
	var ms []QuoteModel
	if err := r.db.WithContext(ctx).Where("symbol = ? AND timestamp >= ? AND timestamp <= ?", symbol, startTime, endTime).Order("timestamp asc").Find(&ms).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.Quote, len(ms))
	for i, m := range ms {
		res[i] = r.toDomain(&m)
	}
	return res, nil
}

func (r *quoteRepositoryImpl) DeleteExpired(ctx context.Context, beforeTime int64) error {
	return r.db.WithContext(ctx).Where("timestamp < ?", beforeTime).Delete(&QuoteModel{}).Error
}

func (r *quoteRepositoryImpl) toDomain(m *QuoteModel) *domain.Quote {
	bp, _ := decimal.NewFromString(m.BidPrice)
	ap, _ := decimal.NewFromString(m.AskPrice)
	bs, _ := decimal.NewFromString(m.BidSize)
	as, _ := decimal.NewFromString(m.AskSize)
	lp, _ := decimal.NewFromString(m.LastPrice)
	ls, _ := decimal.NewFromString(m.LastSize)
	return &domain.Quote{
		Model:     m.Model,
		Symbol:    m.Symbol,
		BidPrice:  bp,
		AskPrice:  ap,
		BidSize:   bs,
		AskSize:   as,
		LastPrice: lp,
		LastSize:  ls,
		Timestamp: m.Timestamp,
		Source:    m.Source,
	}
}

// --- Kline ---

type KlineModel struct {
	gorm.Model
	Symbol           string `gorm:"column:symbol;type:varchar(20);index;not null"`
	Interval         string `gorm:"column:interval;type:varchar(10);not null"`
	OpenTime         int64  `gorm:"column:open_time;type:bigint;not null"`
	Open             string `gorm:"column:open_price;type:decimal(32,18);not null"`
	High             string `gorm:"column:high_price;type:decimal(32,18);not null"`
	Low              string `gorm:"column:low_price;type:decimal(32,18);not null"`
	Close            string `gorm:"column:close_price;type:decimal(32,18);not null"`
	Volume           string `gorm:"column:volume;type:decimal(32,18);not null"`
	CloseTime        int64  `gorm:"column:close_time;type:bigint;not null"`
	QuoteAssetVolume string `gorm:"column:quote_asset_volume;type:decimal(32,18);not null"`
	TradeCount       int64  `gorm:"column:trade_count;type:bigint;not null"`
	TakerBuyBase     string `gorm:"column:taker_buy_base_volume;type:decimal(32,18);not null"`
	TakerBuyQuote    string `gorm:"column:taker_buy_quote_volume;type:decimal(32,18);not null"`
}

func (KlineModel) TableName() string { return "klines" }

type klineRepositoryImpl struct {
	db *gorm.DB
}

func NewKlineRepository(db *gorm.DB) domain.KlineRepository {
	return &klineRepositoryImpl{db: db}
}

func (r *klineRepositoryImpl) Save(ctx context.Context, k *domain.Kline) error {
	m := &KlineModel{
		Model:            k.Model,
		Symbol:           k.Symbol,
		Interval:         k.Interval,
		OpenTime:         k.OpenTime,
		Open:             k.Open.String(),
		High:             k.High.String(),
		Low:              k.Low.String(),
		Close:            k.Close.String(),
		Volume:           k.Volume.String(),
		CloseTime:        k.CloseTime,
		QuoteAssetVolume: k.QuoteAssetVolume.String(),
		TradeCount:       k.TradeCount,
		TakerBuyBase:     k.TakerBuyBaseAssetVolume.String(),
		TakerBuyQuote:    k.TakerBuyQuoteAssetVolume.String(),
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(m).Error
	if err == nil {
		k.Model = m.Model
	}
	return err
}

func (r *klineRepositoryImpl) Get(ctx context.Context, symbol, interval string, startTime, endTime int64) ([]*domain.Kline, error) {
	var ms []KlineModel
	if err := r.db.WithContext(ctx).Where("symbol = ? AND interval = ? AND open_time >= ? AND open_time <= ?", symbol, interval, startTime, endTime).Order("open_time asc").Find(&ms).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.Kline, len(ms))
	for i, m := range ms {
		res[i] = r.toDomain(&m)
	}
	return res, nil
}

func (r *klineRepositoryImpl) GetLatest(ctx context.Context, symbol, interval string, limit int) ([]*domain.Kline, error) {
	var ms []KlineModel
	if err := r.db.WithContext(ctx).Where("symbol = ? AND interval = ?", symbol, interval).Order("open_time desc").Limit(limit).Find(&ms).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.Kline, len(ms))
	for i, m := range ms {
		res[i] = r.toDomain(&m)
	}
	return res, nil
}

func (r *klineRepositoryImpl) DeleteExpired(ctx context.Context, beforeTime int64) error {
	return r.db.WithContext(ctx).Where("open_time < ?", beforeTime).Delete(&KlineModel{}).Error
}

func (r *klineRepositoryImpl) toDomain(m *KlineModel) *domain.Kline {
	o, _ := decimal.NewFromString(m.Open)
	h, _ := decimal.NewFromString(m.High)
	l, _ := decimal.NewFromString(m.Low)
	c, _ := decimal.NewFromString(m.Close)
	v, _ := decimal.NewFromString(m.Volume)
	qav, _ := decimal.NewFromString(m.QuoteAssetVolume)
	tbb, _ := decimal.NewFromString(m.TakerBuyBase)
	tbq, _ := decimal.NewFromString(m.TakerBuyQuote)
	return &domain.Kline{
		Model:                    m.Model,
		Symbol:                   m.Symbol,
		Interval:                 m.Interval,
		OpenTime:                 m.OpenTime,
		Open:                     o,
		High:                     h,
		Low:                      l,
		Close:                    c,
		Volume:                   v,
		CloseTime:                m.CloseTime,
		QuoteAssetVolume:         qav,
		TradeCount:               m.TradeCount,
		TakerBuyBaseAssetVolume:  tbb,
		TakerBuyQuoteAssetVolume: tbq,
	}
}

// --- Trade ---

type TradeModel struct {
	gorm.Model
	TradeID   string `gorm:"column:trade_id;type:varchar(32);uniqueIndex;not null"`
	Symbol    string `gorm:"column:symbol;type:varchar(20);index;not null"`
	Price     string `gorm:"column:price;type:decimal(32,18);not null"`
	Quantity  string `gorm:"column:quantity;type:decimal(32,18);not null"`
	Side      string `gorm:"column:side;type:varchar(10);not null"`
	Timestamp int64  `gorm:"column:timestamp;type:bigint;not null"`
	Source    string `gorm:"column:source;type:varchar(50)"`
}

func (TradeModel) TableName() string { return "trades" }

type tradeRepositoryImpl struct {
	db *gorm.DB
}

func NewTradeRepository(db *gorm.DB) domain.TradeRepository {
	return &tradeRepositoryImpl{db: db}
}

func (r *tradeRepositoryImpl) Save(ctx context.Context, t *domain.Trade) error {
	m := &TradeModel{
		Model:     t.Model,
		TradeID:   t.TradeID,
		Symbol:    t.Symbol,
		Price:     t.Price.String(),
		Quantity:  t.Quantity.String(),
		Side:      t.Side,
		Timestamp: t.Timestamp,
		Source:    t.Source,
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "trade_id"}},
		UpdateAll: true,
	}).Create(m).Error
	if err == nil {
		t.Model = m.Model
	}
	return err
}

func (r *tradeRepositoryImpl) GetHistory(ctx context.Context, symbol string, startTime, endTime int64, limit int) ([]*domain.Trade, error) {
	var ms []TradeModel
	if err := r.db.WithContext(ctx).Where("symbol = ? AND timestamp >= ? AND timestamp <= ?", symbol, startTime, endTime).Order("timestamp asc").Limit(limit).Find(&ms).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.Trade, len(ms))
	for i, m := range ms {
		res[i] = r.toDomain(&m)
	}
	return res, nil
}

func (r *tradeRepositoryImpl) GetLatest(ctx context.Context, symbol string, limit int) ([]*domain.Trade, error) {
	var ms []TradeModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp desc").Limit(limit).Find(&ms).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.Trade, len(ms))
	for i, m := range ms {
		res[i] = r.toDomain(&m)
	}
	return res, nil
}

func (r *tradeRepositoryImpl) DeleteExpired(ctx context.Context, beforeTime int64) error {
	return r.db.WithContext(ctx).Where("timestamp < ?", beforeTime).Delete(&TradeModel{}).Error
}

func (r *tradeRepositoryImpl) toDomain(m *TradeModel) *domain.Trade {
	p, _ := decimal.NewFromString(m.Price)
	q, _ := decimal.NewFromString(m.Quantity)
	return &domain.Trade{
		Model:     m.Model,
		TradeID:   m.TradeID,
		Symbol:    m.Symbol,
		Price:     p,
		Quantity:  q,
		Side:      m.Side,
		Timestamp: m.Timestamp,
		Source:    m.Source,
	}
}
