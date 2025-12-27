// Package mysql 提供了持仓仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PositionModel 持仓数据库模型
type PositionModel struct {
	gorm.Model
	PositionID    string     `gorm:"column:position_id;type:varchar(32);uniqueIndex;not null"`
	UserID        string     `gorm:"column:user_id;type:varchar(32);index;not null"`
	Symbol        string     `gorm:"column:symbol;type:varchar(20);index;not null"`
	Side          string     `gorm:"column:side;type:varchar(10);not null"`
	Quantity      string     `gorm:"column:quantity;type:decimal(32,18);not null"`
	EntryPrice    string     `gorm:"column:entry_price;type:decimal(32,18);not null"`
	CurrentPrice  string     `gorm:"column:current_price;type:decimal(32,18);not null"`
	UnrealizedPnL string     `gorm:"column:unrealized_pnl;type:decimal(32,18);not null"`
	RealizedPnL   string     `gorm:"column:realized_pnl;type:decimal(32,18);not null"`
	OpenedAt      time.Time  `gorm:"column:opened_at;type:datetime;not null"`
	ClosedAt      *time.Time `gorm:"column:closed_at;type:datetime"`
	Status        string     `gorm:"column:status;type:varchar(20);index;not null"`
}

// TableName 指定表名
func (PositionModel) TableName() string {
	return "positions"
}

// positionRepositoryImpl 是 domain.PositionRepository 接口的 GORM 实现。
type positionRepositoryImpl struct {
	db *gorm.DB
}

// NewPositionRepository 创建持仓仓储实例
func NewPositionRepository(db *gorm.DB) domain.PositionRepository {
	return &positionRepositoryImpl{
		db: db,
	}
}

// Save 实现 domain.PositionRepository.Save
func (r *positionRepositoryImpl) Save(ctx context.Context, position *domain.Position) error {
	model := r.fromDomain(position)
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "position_id"}},
		UpdateAll: true,
	}).Create(model).Error

	if err != nil {
		logging.Error(ctx, "position_repository.Save failed", "position_id", position.PositionID, "error", err)
		return fmt.Errorf("failed to save position: %w", err)
	}

	position.Model = model.Model
	return nil
}

// Get 实现 domain.PositionRepository.Get
func (r *positionRepositoryImpl) Get(ctx context.Context, positionID string) (*domain.Position, error) {
	var model PositionModel
	if err := r.db.WithContext(ctx).Where("position_id = ?", positionID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "position_repository.Get failed", "position_id", positionID, "error", err)
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	return r.toDomain(&model), nil
}

// GetByUser 实现 domain.PositionRepository.GetByUser
func (r *positionRepositoryImpl) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Position, int64, error) {
	var models []PositionModel
	var total int64
	db := r.db.WithContext(ctx).Model(&PositionModel{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		logging.Error(ctx, "position_repository.GetByUser failed", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("failed to get positions by user: %w", err)
	}

	positions := make([]*domain.Position, len(models))
	for i, m := range models {
		positions[i] = r.toDomain(&m)
	}
	return positions, total, nil
}

// GetBySymbol 实现 domain.PositionRepository.GetBySymbol
func (r *positionRepositoryImpl) GetBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*domain.Position, int64, error) {
	var models []PositionModel
	var total int64
	db := r.db.WithContext(ctx).Model(&PositionModel{}).Where("symbol = ?", symbol)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		logging.Error(ctx, "position_repository.GetBySymbol failed", "symbol", symbol, "error", err)
		return nil, 0, fmt.Errorf("failed to get positions by symbol: %w", err)
	}

	positions := make([]*domain.Position, len(models))
	for i, m := range models {
		positions[i] = r.toDomain(&m)
	}
	return positions, total, nil
}

// Update 实现 domain.PositionRepository.Update
func (r *positionRepositoryImpl) Update(ctx context.Context, position *domain.Position) error {
	model := r.fromDomain(position)
	err := r.db.WithContext(ctx).Model(&PositionModel{}).Where("position_id = ?", position.PositionID).Updates(model).Error
	if err != nil {
		logging.Error(ctx, "position_repository.Update failed", "position_id", position.PositionID, "error", err)
		return fmt.Errorf("failed to update position: %w", err)
	}
	return nil
}

// Close 实现 domain.PositionRepository.Close
func (r *positionRepositoryImpl) Close(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	now := time.Now()
	err := r.db.WithContext(ctx).Model(&PositionModel{}).Where("position_id = ?", positionID).Updates(map[string]interface{}{
		"status":        "CLOSED",
		"closed_at":     &now,
		"current_price": closePrice.String(),
	}).Error

	if err != nil {
		logging.Error(ctx, "position_repository.Close failed", "position_id", positionID, "error", err)
		return fmt.Errorf("failed to close position: %w", err)
	}
	return nil
}

func (r *positionRepositoryImpl) fromDomain(p *domain.Position) *PositionModel {
	return &PositionModel{
		Model:         p.Model,
		PositionID:    p.PositionID,
		UserID:        p.UserID,
		Symbol:        p.Symbol,
		Side:          p.Side,
		Quantity:      p.Quantity.String(),
		EntryPrice:    p.EntryPrice.String(),
		CurrentPrice:  p.CurrentPrice.String(),
		UnrealizedPnL: p.UnrealizedPnL.String(),
		RealizedPnL:   p.RealizedPnL.String(),
		OpenedAt:      p.OpenedAt,
		ClosedAt:      p.ClosedAt,
		Status:        p.Status,
	}
}

func (r *positionRepositoryImpl) toDomain(m *PositionModel) *domain.Position {
	qty, _ := decimal.NewFromString(m.Quantity)
	entry, _ := decimal.NewFromString(m.EntryPrice)
	current, _ := decimal.NewFromString(m.CurrentPrice)
	unrealized, _ := decimal.NewFromString(m.UnrealizedPnL)
	realized, _ := decimal.NewFromString(m.RealizedPnL)

	return &domain.Position{
		Model:         m.Model,
		PositionID:    m.PositionID,
		UserID:        m.UserID,
		Symbol:        m.Symbol,
		Side:          m.Side,
		Quantity:      qty,
		EntryPrice:    entry,
		CurrentPrice:  current,
		UnrealizedPnL: unrealized,
		RealizedPnL:   realized,
		OpenedAt:      m.OpenedAt,
		ClosedAt:      m.ClosedAt,
		Status:        m.Status,
	}
}
