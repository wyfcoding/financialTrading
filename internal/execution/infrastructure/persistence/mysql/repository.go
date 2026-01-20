package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type TradeRepository struct {
	db *gorm.DB
}

func NewTradeRepository(db *gorm.DB) *TradeRepository {
	return &TradeRepository{db: db}
}

func (r *TradeRepository) Save(ctx context.Context, t *domain.Trade) error {
	// Directly save domain object as it has GORM tags
	return r.getDB(ctx).Create(t).Error
}

func (r *TradeRepository) Get(ctx context.Context, id string) (*domain.Trade, error) {
	var trade domain.Trade
	if err := r.getDB(ctx).Where("trade_id = ?", id).First(&trade).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &trade, nil
}

func (r *TradeRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Trade, error) {
	var trade domain.Trade
	if err := r.getDB(ctx).Where("order_id = ?", orderID).First(&trade).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &trade, nil
}

func (r *TradeRepository) List(ctx context.Context, userID string) ([]*domain.Trade, error) {
	var trades []*domain.Trade
	if err := r.getDB(ctx).Where("user_id = ?", userID).Find(&trades).Error; err != nil {
		return nil, err
	}
	return trades, nil
}

func (r *TradeRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}

type AlgoOrderRepository struct {
	db *gorm.DB
}

func NewAlgoOrderRepository(db *gorm.DB) *AlgoOrderRepository {
	return &AlgoOrderRepository{db: db}
}

func (r *AlgoOrderRepository) Save(ctx context.Context, o *domain.AlgoOrder) error {
	// Check if exists using AlgoID
	var existing domain.AlgoOrder
	if err := r.getDB(ctx).Where("algo_id = ?", o.AlgoID).First(&existing).Error; err == nil {
		o.ID = existing.ID // Preserve GORM ID
		return r.getDB(ctx).Save(o).Error
	}
	return r.getDB(ctx).Create(o).Error
}

func (r *AlgoOrderRepository) Get(ctx context.Context, algoID string) (*domain.AlgoOrder, error) {
	var order domain.AlgoOrder
	if err := r.getDB(ctx).Where("algo_id = ?", algoID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *AlgoOrderRepository) ListActive(ctx context.Context) ([]*domain.AlgoOrder, error) {
	var orders []*domain.AlgoOrder
	// Assuming ACTIVE status is generic string or defined in domain.
	// In Step 2866 it was string "PENDING", "ACTIVE".
	if err := r.getDB(ctx).Where("status = ?", "ACTIVE").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *AlgoOrderRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
