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
	Symbol      string `gorm:"column:symbol;type:varchar(20);index:idx_symbol_time;not null"`
	BuyOrderID  string `gorm:"column:buy_order_id;type:varchar(32);index;not null"`
	SellOrderID string `gorm:"column:sell_order_id;type:varchar(32);index;not null"`
	Price       string `gorm:"column:price;type:decimal(32,18);not null"`
	Quantity    string `gorm:"column:quantity;type:decimal(32,18);not null"`
	ExecutedAt  int64  `gorm:"column:executed_at;type:bigint;index:idx_symbol_time"`
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

// getDBFromContext 尝试从 Context 获取事务 DB，否则返回默认 DB
func (r *matchingRepositoryImpl) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value("tx_db").(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
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
	return r.getDB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "trade_id"}},
		UpdateAll: true,
	}).Create(m).Error
}

func (r *matchingRepositoryImpl) GetTradeHistory(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error) {
	var models []TradeModel
	if err := r.getDB(ctx).Where("symbol = ?", symbol).Order("executed_at desc").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	res := make([]*algorithm.Trade, len(models))
	for i, m := range models {
		t, err := r.tradeToAlgorithm(&m)
		if err != nil {
			return nil, err
		}
		res[i] = t
	}
	return res, nil
}

func (r *matchingRepositoryImpl) GetLatestTrades(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error) {
	return r.GetTradeHistory(ctx, symbol, limit)
}

// OrderBookRepository methods
func (r *matchingRepositoryImpl) SaveSnapshot(ctx context.Context, s *domain.OrderBookSnapshot) error {
	bids, err := json.Marshal(s.Bids)
	if err != nil {
		return err
	}
	asks, err := json.Marshal(s.Asks)
	if err != nil {
		return err
	}
	m := &OrderBookSnapshotModel{
		Model:     s.Model,
		Symbol:    s.Symbol,
		Bids:      string(bids),
		Asks:      string(asks),
		Timestamp: s.Timestamp,
	}
	err = r.getDB(ctx).Create(m).Error
	if err == nil {
		s.Model = m.Model
	}
	return err
}

func (r *matchingRepositoryImpl) GetLatestOrderBook(ctx context.Context, symbol string) (*domain.OrderBookSnapshot, error) {
	var m OrderBookSnapshotModel
	if err := r.getDB(ctx).Where("symbol = ?", symbol).Order("timestamp desc").First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.snapshotToDomain(&m)
}

func (r *matchingRepositoryImpl) tradeToAlgorithm(m *TradeModel) (*algorithm.Trade, error) {
	price, err := decimal.NewFromString(m.Price)
	if err != nil {
		return nil, err
	}
	qty, err := decimal.NewFromString(m.Quantity)
	if err != nil {
		return nil, err
	}
	return &algorithm.Trade{
		TradeID:     m.TradeID,
		Symbol:      m.Symbol,
		BuyOrderID:  m.BuyOrderID,
		SellOrderID: m.SellOrderID,
		Price:       price,
		Quantity:    qty,
		Timestamp:   m.ExecutedAt,
	}, nil
}

func (r *matchingRepositoryImpl) snapshotToDomain(m *OrderBookSnapshotModel) (*domain.OrderBookSnapshot, error) {
	var bids, asks []*domain.OrderBookLevel
	if err := json.Unmarshal([]byte(m.Bids), &bids); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(m.Asks), &asks); err != nil {
		return nil, err
	}
	return &domain.OrderBookSnapshot{
		Model:     m.Model,
		Symbol:    m.Symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: m.Timestamp,
	}, nil
}
