package persistence

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type positionRepository struct {
	db *gorm.DB
}

// NewPositionRepository 创建并返回一个新的 positionRepository 实例。
func NewPositionRepository(db *gorm.DB) domain.PositionRepository {
	return &positionRepository{db: db}
}

func (r *positionRepository) Save(ctx context.Context, position *domain.Position) error {
	db := r.getDB(ctx)
	if position.ID == 0 {
		return db.Create(position).Error
	}
	return db.Save(position).Error
}

func (r *positionRepository) GetByUserSymbol(ctx context.Context, userID, symbol string) (*domain.Position, error) {
	var pos domain.Position
	err := r.getDB(ctx).Preload("Lots").Where("user_id = ? AND symbol = ?", userID, symbol).First(&pos).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pos, nil
}

func (r *positionRepository) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Position, int64, error) {
	var positions []*domain.Position
	var total int64
	db := r.getDB(ctx).Model(&domain.Position{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Preload("Lots").Limit(limit).Offset(offset).Find(&positions).Error; err != nil {
		return nil, 0, err
	}
	return positions, total, nil
}

func (r *positionRepository) GetBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*domain.Position, int64, error) {
	var positions []*domain.Position
	var total int64
	db := r.getDB(ctx).Model(&domain.Position{}).Where("symbol = ?", symbol)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Preload("Lots").Limit(limit).Offset(offset).Find(&positions).Error; err != nil {
		return nil, 0, err
	}
	return positions, total, nil
}

func (r *positionRepository) ExecWithBarrier(ctx context.Context, barrier any, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *positionRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}

// 补充接口缺失的方法实现
func (r *positionRepository) Get(ctx context.Context, positionID string) (*domain.Position, error) {
	var pos domain.Position
	err := r.getDB(ctx).Preload("Lots").First(&pos, positionID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pos, nil
}

func (r *positionRepository) Update(ctx context.Context, position *domain.Position) error {
	return r.Save(ctx, position)
}

func (r *positionRepository) Close(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	// 简单实现：将仓位清零
	return r.getDB(ctx).Model(&domain.Position{}).Where("id = ?", positionID).Updates(map[string]interface{}{
		"quantity":            0,
		"average_entry_price": 0,
	}).Error
}
