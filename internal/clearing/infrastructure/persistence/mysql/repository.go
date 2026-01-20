package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type SettlementRepository struct {
	db *gorm.DB
}

func NewSettlementRepository(db *gorm.DB) *SettlementRepository {
	return &SettlementRepository{db: db}
}

func (r *SettlementRepository) Save(ctx context.Context, s *domain.Settlement) error {
	po := &SettlementPO{}
	po.FromDomain(s)

	db := r.getDB(ctx)

	// Create or Update
	var existing SettlementPO
	if err := db.Where("settlement_id = ?", s.ID).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return db.Create(po).Error
		}
		return err
	}

	// Update
	// Note: gorm.Model contains ID uint, so we shouldn't overwrite it if it exists.
	// But `po` here is fresh, so ID is 0. Updates with map or struct that has 0 id might be tricky if not careful.
	// Let's use Updates on query.
	return db.Model(&SettlementPO{}).Where("settlement_id = ?", s.ID).Updates(po).Error
}

func (r *SettlementRepository) Get(ctx context.Context, id string) (*domain.Settlement, error) {
	var po SettlementPO
	if err := r.getDB(ctx).Where("settlement_id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return po.ToDomain(), nil
}

func (r *SettlementRepository) GetByTradeID(ctx context.Context, tradeID string) (*domain.Settlement, error) {
	var po SettlementPO
	if err := r.getDB(ctx).Where("trade_id = ?", tradeID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return po.ToDomain(), nil
}

func (r *SettlementRepository) List(ctx context.Context, limit int) ([]*domain.Settlement, error) {
	var pos []SettlementPO
	if err := r.getDB(ctx).Limit(limit).Order("id desc").Find(&pos).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.Settlement, len(pos))
	for i, po := range pos {
		res[i] = po.ToDomain()
	}
	return res, nil
}

func (r *SettlementRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
