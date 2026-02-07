package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type tradeRepository struct {
	db *gorm.DB
}

func NewTradeRepository(db *gorm.DB) domain.TradeRepository {
	return &tradeRepository{db: db}
}

// --- tx helpers ---

func (r *tradeRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *tradeRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *tradeRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *tradeRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *tradeRepository) Save(ctx context.Context, trade *domain.Trade) error {
	model := toTradeModel(trade)
	if model == nil {
		return nil
	}
	db := r.getDB(ctx).WithContext(ctx)
	if model.ID == 0 {
		if err := db.Create(model).Error; err != nil {
			return err
		}
		trade.ID = model.ID
		trade.CreatedAt = model.CreatedAt
		trade.UpdatedAt = model.UpdatedAt
		return nil
	}
	return db.Model(&TradeModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"trade_id":      model.TradeID,
			"buy_order_id":  model.BuyOrderID,
			"sell_order_id": model.SellOrderID,
			"symbol":        model.Symbol,
			"price":         model.Price,
			"quantity":      model.Quantity,
			"timestamp":     model.Timestamp,
		}).Error
}

func (r *tradeRepository) GetLatestTrades(ctx context.Context, symbol string, limit int) ([]*domain.Trade, error) {
	var models []*TradeModel
	if err := r.getDB(ctx).WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("timestamp desc").
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, err
	}
	trades := make([]*domain.Trade, len(models))
	for i, m := range models {
		trades[i] = toTrade(m)
	}
	return trades, nil
}

func (r *tradeRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}

type orderBookRepository struct {
	db *gorm.DB
}

func NewOrderBookRepository(db *gorm.DB) domain.OrderBookRepository {
	return &orderBookRepository{db: db}
}

func (r *orderBookRepository) SaveSnapshot(ctx context.Context, snapshot *domain.OrderBookSnapshot) error {
	model, err := toSnapshotModel(snapshot)
	if err != nil {
		return err
	}
	if model == nil {
		return nil
	}
	db := r.db.WithContext(ctx)
	if model.ID == 0 {
		if err := db.Create(model).Error; err != nil {
			return err
		}
		snapshot.ID = model.ID
		snapshot.CreatedAt = model.CreatedAt
		snapshot.UpdatedAt = model.UpdatedAt
		return nil
	}
	return db.Model(&OrderBookSnapshotModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"symbol":    model.Symbol,
			"bids":      model.BidsJSON,
			"asks":      model.AsksJSON,
			"timestamp": model.Timestamp,
		}).Error
}
