// Package repository 包含仓储实现
package repository

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/internal/matching-engine/domain"
	"github.com/wyfcoding/pkg/algorithm"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
)

// TradeModel 成交记录数据库模型
// 对应数据库中的 trades 表
type TradeModel struct {
	gorm.Model
	// 成交 ID
	TradeID string `gorm:"column:trade_id;type:varchar(50);uniqueIndex;not null;comment:成交ID" json:"trade_id"`
	// 交易对
	Symbol string `gorm:"column:symbol;type:varchar(50);index;not null;comment:交易对" json:"symbol"`
	// 买方订单 ID
	BuyOrderID string `gorm:"column:buy_order_id;type:varchar(50);index;not null;comment:买方订单ID" json:"buy_order_id"`
	// 卖方订单 ID
	SellOrderID string `gorm:"column:sell_order_id;type:varchar(50);index;not null;comment:卖方订单ID" json:"sell_order_id"`
	// 成交价格
	Price string `gorm:"column:price;type:decimal(20,8);not null;comment:成交价格" json:"price"`
	// 成交数量
	Quantity string `gorm:"column:quantity;type:decimal(20,8);not null" json:"quantity"`
	// 时间戳
	Timestamp int64 `gorm:"column:timestamp;type:bigint;index;not null" json:"timestamp"`
}

// TableName 指定表名
func (TradeModel) TableName() string {
	return "trades"
}

// TradeRepositoryImpl 成交记录仓储实现
type TradeRepositoryImpl struct {
	db *gorm.DB
}

// NewTradeRepository 创建成交记录仓储
func NewTradeRepository(database *gorm.DB) domain.TradeRepository {
	return &TradeRepositoryImpl{
		db: database,
	}
}

// Save 保存成交记录
func (tr *TradeRepositoryImpl) Save(ctx context.Context, trade *algorithm.Trade) error {
	model := &TradeModel{
		TradeID:     trade.TradeID,
		Symbol:      trade.Symbol,
		BuyOrderID:  trade.BuyOrderID,
		SellOrderID: trade.SellOrderID,
		Price:       trade.Price.String(),
		Quantity:    trade.Quantity.String(),
		Timestamp:   trade.Timestamp,
	}

	if err := tr.db.WithContext(ctx).Create(model).Error; err != nil {
		logging.Error(ctx, "Failed to save trade",
			"trade_id", trade.TradeID,
			"error", err,
		)
		return fmt.Errorf("failed to save trade: %w", err)
	}

	return nil
}

// GetHistory 获取成交历史
func (tr *TradeRepositoryImpl) GetHistory(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error) {
	var models []TradeModel

	if err := tr.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp DESC").Limit(limit).Find(&models).Error; err != nil {
		logging.Error(ctx, "Failed to get trade history",
			"symbol", symbol,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get trade history: %w", err)
	}

	trades := make([]*algorithm.Trade, 0, len(models))
	for _, model := range models {
		price, _ := parseDecimal(model.Price)
		quantity, _ := parseDecimal(model.Quantity)
		trades = append(trades, &algorithm.Trade{
			TradeID:     model.TradeID,
			Symbol:      model.Symbol,
			BuyOrderID:  model.BuyOrderID,
			SellOrderID: model.SellOrderID,
			Price:       price,
			Quantity:    quantity,
			Timestamp:   model.Timestamp,
		})
	}

	return trades, nil
}

// GetLatest 获取最新成交
func (tr *TradeRepositoryImpl) GetLatest(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error) {
	return tr.GetHistory(ctx, symbol, limit)
}

// OrderBookRepositoryImpl 订单簿仓储实现
type OrderBookRepositoryImpl struct {
	db *gorm.DB
}

// NewOrderBookRepository 创建订单簿仓储
func NewOrderBookRepository(database *gorm.DB) domain.OrderBookRepository {
	return &OrderBookRepositoryImpl{
		db: database,
	}
}

// SaveSnapshot 保存订单簿快照
func (obr *OrderBookRepositoryImpl) SaveSnapshot(ctx context.Context, snapshot *domain.OrderBookSnapshot) error {
	// 实现待补充：保存订单簿快照到数据库或缓存
	logging.Debug(ctx, "Order book snapshot saved",
		"symbol", snapshot.Symbol,
		"bids_count", len(snapshot.Bids),
		"asks_count", len(snapshot.Asks),
	)
	return nil
}

// GetLatest 获取最新订单簿
func (obr *OrderBookRepositoryImpl) GetLatest(ctx context.Context, symbol string) (*domain.OrderBookSnapshot, error) {
	// 实现待补充：从数据库或缓存获取最新订单簿
	return &domain.OrderBookSnapshot{
		Symbol:    symbol,
		Bids:      make([]*domain.OrderBookLevel, 0),
		Asks:      make([]*domain.OrderBookLevel, 0),
		Timestamp: 0,
	}, nil
}

// parseDecimal 解析十进制字符串
func parseDecimal(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}
