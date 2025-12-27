// Package mysql 提供了撮合引擎成交记录与订单簿仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/pkg/algorithm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TradeModel 成交记录数据库模型
type TradeModel struct {
	gorm.Model
	TradeID     string `gorm:"column:trade_id;type:varchar(32);uniqueIndex;not null"`
	Symbol      string `gorm:"column:symbol;type:varchar(20);index;not null"`
	BuyOrderID  string `gorm:"column:buy_order_id;type:varchar(32);index;not null"`
	SellOrderID string `gorm:"column:sell_order_id;type:varchar(32);index;not null"`
	Price       string `gorm:"column:price;type:decimal(32,18);not null"`
	Quantity    string `gorm:"column:quantity;type:decimal(32,18);not null"`
	ExecutedAt  int64  `gorm:"column:executed_at;type:bigint"`
}

func (TradeModel) TableName() string { return "matching_trades" }

// OrderBookSnapshotModel 订单簿快照数据库模型
type OrderBookSnapshotModel struct {
	gorm.Model
	Symbol    string `gorm:"column:symbol;type:varchar(20);index;not null"`
	Bids      string `gorm:"column:bids;type:text"`
	Asks      string `gorm:"column:asks;type:text"`
	Timestamp int64  `gorm:"column:timestamp;type:bigint"`
}

func (OrderBookSnapshotModel) TableName() string { return "matching_order_book_snapshots" }

// matchingRepositoryImpl 撮合仓储实现
type matchingRepositoryImpl struct {
	db *gorm.DB
}

func NewMatchingRepository(db *gorm.DB) (domain.TradeRepository, domain.OrderBookRepository) {
	impl := &matchingRepositoryImpl{db: db}
	return impl, impl
}

// TradeRepository methods
func (r *matchingRepositoryImpl) Save(ctx context.Context, t *algorithm.Trade) error {
	m := &TradeModel{
		TradeID:     t.TradeID,
		Symbol:      t.Symbol,
		BuyOrderID:  t.BuyOrderID,
		SellOrderID: t.SellOrderID,
		Price:       t.Price.String(),
		Quantity:    t.Quantity.String(),
		ExecutedAt:  t.Timestamp,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "trade_id"}},
		UpdateAll: true,
	}).Create(m).Error
}

func (r *matchingRepositoryImpl) GetTradeHistory(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error) {
	var models []TradeModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("executed_at desc").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	res := make([]*algorithm.Trade, len(models))
	for i, m := range models {
		res[i] = r.tradeToAlgorithm(&m)
	}
	return res, nil
}

func (r *matchingRepositoryImpl) GetLatestTrades(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error) {
	return r.GetTradeHistory(ctx, symbol, limit)
}

// OrderBookRepository methods
func (r *matchingRepositoryImpl) SaveSnapshot(ctx context.Context, s *domain.OrderBookSnapshot) error {
	bids, _ := json.Marshal(s.Bids)
	asks, _ := json.Marshal(s.Asks)
	m := &OrderBookSnapshotModel{
		Model:     s.Model,
		Symbol:    s.Symbol,
		Bids:      string(bids),
		Asks:      string(asks),
		Timestamp: s.Timestamp,
	}
	err := r.db.WithContext(ctx).Create(m).Error
	if err == nil {
		s.Model = m.Model
	}
	return err
}

func (r *matchingRepositoryImpl) GetLatestOrderBook(ctx context.Context, symbol string) (*domain.OrderBookSnapshot, error) {
	var m OrderBookSnapshotModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp desc").First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.snapshotToDomain(&m), nil
}

func (r *matchingRepositoryImpl) tradeToAlgorithm(m *TradeModel) *algorithm.Trade {
	price, _ := decimal.NewFromString(m.Price)
	qty, _ := decimal.NewFromString(m.Quantity)
	return &algorithm.Trade{
		TradeID:     m.TradeID,
		Symbol:      m.Symbol,
		BuyOrderID:  m.BuyOrderID,
		SellOrderID: m.SellOrderID,
		Price:       price,
		Quantity:    qty,
		Timestamp:   m.ExecutedAt,
	}
}

func (r *matchingRepositoryImpl) snapshotToDomain(m *OrderBookSnapshotModel) *domain.OrderBookSnapshot {
	var bids, asks []*domain.OrderBookLevel
	json.Unmarshal([]byte(m.Bids), &bids)
	json.Unmarshal([]byte(m.Asks), &asks)
	return &domain.OrderBookSnapshot{
		Model:     m.Model,
		Symbol:    m.Symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: m.Timestamp,
	}
}
