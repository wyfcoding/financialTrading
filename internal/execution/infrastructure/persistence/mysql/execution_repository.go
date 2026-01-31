package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type tradeRepository struct {
	db *gorm.DB
}

// NewTradeRepository 创建并返回一个新的 tradeRepository 实例。
func NewTradeRepository(db *gorm.DB) domain.TradeRepository {
	return &tradeRepository{db: db}
}

func (r *tradeRepository) Save(ctx context.Context, t *domain.Trade) error {
	return r.getDB(ctx).Create(t).Error
}

func (r *tradeRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Trade, error) {
	var trade domain.Trade
	if err := r.getDB(ctx).Where("order_id = ?", orderID).First(&trade).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &trade, nil
}

func (r *tradeRepository) List(ctx context.Context, userID string) ([]*domain.Trade, error) {
	var trades []*domain.Trade
	if err := r.getDB(ctx).Where("user_id = ?", userID).Find(&trades).Error; err != nil {
		return nil, err
	}
	return trades, nil
}

func (r *tradeRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}

type algoOrderRepository struct {
	db *gorm.DB
}

// NewAlgoOrderRepository 创建并返回一个新的 algoOrderRepository 实例。
func NewAlgoOrderRepository(db *gorm.DB) domain.AlgoOrderRepository {
	return &algoOrderRepository{db: db}
}

func (r *algoOrderRepository) Save(ctx context.Context, o *domain.AlgoOrder) error {
	var existing domain.AlgoOrder
	if err := r.getDB(ctx).Where("algo_id = ?", o.AlgoID).First(&existing).Error; err == nil {
		o.ID = existing.ID
		return r.getDB(ctx).Save(o).Error
	}
	return r.getDB(ctx).Create(o).Error
}

func (r *algoOrderRepository) Get(ctx context.Context, algoID string) (*domain.AlgoOrder, error) {
	var order domain.AlgoOrder
	if err := r.getDB(ctx).Where("algo_id = ?", algoID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *algoOrderRepository) ListActive(ctx context.Context) ([]*domain.AlgoOrder, error) {
	var orders []*domain.AlgoOrder
	if err := r.getDB(ctx).Where("status = ?", "ACTIVE").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *algoOrderRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
