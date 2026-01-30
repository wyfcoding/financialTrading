package persistence

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type settlementRepository struct {
	db *gorm.DB
}

// NewSettlementRepository 创建并返回一个新的 settlementRepository 实例。
func NewSettlementRepository(db *gorm.DB) domain.SettlementRepository {
	return &settlementRepository{db: db}
}

func (r *settlementRepository) Save(ctx context.Context, s *domain.Settlement) error {
	return r.getDB(ctx).Save(s).Error
}

func (r *settlementRepository) Get(ctx context.Context, id string) (*domain.Settlement, error) {
	var s domain.Settlement
	if err := r.getDB(ctx).Where("settlement_id = ?", id).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *settlementRepository) GetByTradeID(ctx context.Context, tradeID string) (*domain.Settlement, error) {
	var s domain.Settlement
	if err := r.getDB(ctx).Where("trade_id = ?", tradeID).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *settlementRepository) List(ctx context.Context, limit int) ([]*domain.Settlement, error) {
	var list []*domain.Settlement
	if err := r.getDB(ctx).Limit(limit).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *settlementRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
