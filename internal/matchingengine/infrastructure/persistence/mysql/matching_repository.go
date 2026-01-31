package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"gorm.io/gorm"
)

type tradeRepository struct {
	db *gorm.DB
}

func NewTradeRepository(db *gorm.DB) domain.TradeRepository {
	return &tradeRepository{db: db}
}

func (r *tradeRepository) Save(ctx context.Context, trade *domain.Trade) error {
	return r.db.WithContext(ctx).Create(trade).Error
}

func (r *tradeRepository) GetLatestTrades(ctx context.Context, symbol string, limit int) ([]*domain.Trade, error) {
	var trades []*domain.Trade
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp desc").Limit(limit).Find(&trades).Error; err != nil {
		return nil, err
	}
	return trades, nil
}

type orderBookRepository struct {
	db *gorm.DB
}

func NewOrderBookRepository(db *gorm.DB) domain.OrderBookRepository {
	return &orderBookRepository{db: db}
}

func (r *orderBookRepository) SaveSnapshot(ctx context.Context, snapshot *domain.OrderBookSnapshot) error {
	return r.db.WithContext(ctx).Save(snapshot).Error
}
