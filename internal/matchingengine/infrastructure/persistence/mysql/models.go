package mysql

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"gorm.io/gorm"
)

// OrderModel MySQL 订单表映射
type OrderModel struct {
	gorm.Model
	OrderID   string    `gorm:"column:order_id;type:varchar(32);uniqueIndex;not null;comment:订单ID"`
	Symbol    string    `gorm:"column:symbol;type:varchar(20);index;not null;comment:标的"`
	Side      int       `gorm:"column:side;type:tinyint;not null;comment:方向"`
	Type      int       `gorm:"column:type;type:tinyint;not null;comment:类型"`
	Price     float64   `gorm:"column:price;type:double;not null;comment:价格"`
	Quantity  float64   `gorm:"column:quantity;type:double;not null;comment:数量"`
	Timestamp time.Time `gorm:"column:timestamp;not null;comment:时间"`
	FilledQty float64   `gorm:"column:filled_qty;type:double;default:0;comment:已成交量"`
}

func (OrderModel) TableName() string { return "matching_orders" }

// TradeModel MySQL 成交表映射
type TradeModel struct {
	gorm.Model
	TradeID     string    `gorm:"column:trade_id;type:varchar(32);uniqueIndex;not null;comment:成交ID"`
	BuyOrderID  string    `gorm:"column:buy_order_id;type:varchar(32);index;not null"`
	SellOrderID string    `gorm:"column:sell_order_id;type:varchar(32);index;not null"`
	Symbol      string    `gorm:"column:symbol;type:varchar(20);index;not null"`
	Price       float64   `gorm:"column:price;type:double;not null"`
	Quantity    float64   `gorm:"column:quantity;type:double;not null"`
	Timestamp   time.Time `gorm:"column:timestamp;not null"`
}

func (TradeModel) TableName() string { return "matching_trades" }

// OrderBookSnapshotModel MySQL 订单簿快照表映射
type OrderBookSnapshotModel struct {
	gorm.Model
	Symbol    string `gorm:"column:symbol;type:varchar(20);index;not null"`
	BidsJSON  string `gorm:"column:bids;type:json"`
	AsksJSON  string `gorm:"column:asks;type:json"`
	Timestamp int64  `gorm:"column:timestamp;not null"`
}

func (OrderBookSnapshotModel) TableName() string { return "order_book_snapshots" }

// mapping helpers

func toTradeModel(t *domain.Trade) *TradeModel {
	if t == nil {
		return nil
	}
	return &TradeModel{
		Model: gorm.Model{
			ID:        t.ID,
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		},
		TradeID:     t.TradeID,
		BuyOrderID:  t.BuyOrderID,
		SellOrderID: t.SellOrderID,
		Symbol:      t.Symbol,
		Price:       t.Price,
		Quantity:    t.Quantity,
		Timestamp:   t.Timestamp,
	}
}

func toTrade(m *TradeModel) *domain.Trade {
	if m == nil {
		return nil
	}
	return &domain.Trade{
		ID:          m.ID,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		TradeID:     m.TradeID,
		BuyOrderID:  m.BuyOrderID,
		SellOrderID: m.SellOrderID,
		Symbol:      m.Symbol,
		Price:       m.Price,
		Quantity:    m.Quantity,
		Timestamp:   m.Timestamp,
	}
}

func toSnapshotModel(s *domain.OrderBookSnapshot) (*OrderBookSnapshotModel, error) {
	if s == nil {
		return nil, nil
	}
	bids, err := json.Marshal(s.Bids)
	if err != nil {
		return nil, err
	}
	asks, err := json.Marshal(s.Asks)
	if err != nil {
		return nil, err
	}
	return &OrderBookSnapshotModel{
		Model: gorm.Model{
			ID:        s.ID,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		},
		Symbol:    s.Symbol,
		BidsJSON:  string(bids),
		AsksJSON:  string(asks),
		Timestamp: s.Timestamp,
	}, nil
}

func toSnapshot(m *OrderBookSnapshotModel) (*domain.OrderBookSnapshot, error) {
	if m == nil {
		return nil, nil
	}
	var bids []*domain.OrderBookLevel
	var asks []*domain.OrderBookLevel
	if err := json.Unmarshal([]byte(m.BidsJSON), &bids); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(m.AsksJSON), &asks); err != nil {
		return nil, err
	}
	return &domain.OrderBookSnapshot{
		ID:        m.ID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		Symbol:    m.Symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: m.Timestamp,
	}, nil
}

// toLevel ensures decimal fields are set on json decode
func toLevel(price, quantity float64) *domain.OrderBookLevel {
	return &domain.OrderBookLevel{
		Price:    decimal.NewFromFloat(price),
		Quantity: decimal.NewFromFloat(quantity),
	}
}
