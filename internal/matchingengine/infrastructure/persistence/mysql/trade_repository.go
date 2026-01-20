package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/pkg/algorithm/types"
	"gorm.io/gorm"
)

type tradeRepository struct {
	db *gorm.DB
}

func NewTradeRepository(db *gorm.DB) domain.TradeRepository {
	return &tradeRepository{db: db}
}

func (r *tradeRepository) Save(ctx context.Context, trade *types.Trade) error {
	// Map types.Trade to GORM model if needed, or save directly if compatible
	// Assuming a TradeModel exists or using types.Trade if GORM tags present.
	return r.db.WithContext(ctx).Create(trade).Error
}

func (r *tradeRepository) FindByOrderID(ctx context.Context, orderID string) ([]*types.Trade, error) {
	var trades []*types.Trade
	err := r.db.WithContext(ctx).Where("buy_order_id = ? OR sell_order_id = ?", orderID, orderID).Find(&trades).Error
	return trades, err
}

func (r *tradeRepository) GetLatestTrades(ctx context.Context, symbol string, limit int) ([]*types.Trade, error) {
	var trades []*types.Trade
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp desc").Limit(limit).Find(&trades).Error
	return trades, err
}
