// Package repository 包含仓储实现
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/wyfcoding/financialTrading/internal/position/domain"
	"github.com/wyfcoding/financialTrading/pkg/db"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// PositionModel 持仓数据库模型
type PositionModel struct {
	gorm.Model
	// 持仓 ID
	PositionID string `gorm:"column:position_id;type:varchar(50);uniqueIndex;not null" json:"position_id"`
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(50);index;not null" json:"user_id"`
	// 交易对
	Symbol string `gorm:"column:symbol;type:varchar(50);index;not null" json:"symbol"`
	// 买卖方向
	Side string `gorm:"column:side;type:varchar(10);not null" json:"side"`
	// 持仓数量
	Quantity string `gorm:"column:quantity;type:decimal(20,8);not null" json:"quantity"`
	// 开仓价格
	EntryPrice string `gorm:"column:entry_price;type:decimal(20,8);not null" json:"entry_price"`
	// 当前价格
	CurrentPrice string `gorm:"column:current_price;type:decimal(20,8);not null" json:"current_price"`
	// 未实现盈亏
	UnrealizedPnL string `gorm:"column:unrealized_pnl;type:decimal(20,8);not null" json:"unrealized_pnl"`
	// 已实现盈亏
	RealizedPnL string `gorm:"column:realized_pnl;type:decimal(20,8);not null" json:"realized_pnl"`
	// 开仓时间
	OpenedAt time.Time `gorm:"column:opened_at;type:datetime;not null" json:"opened_at"`
	// 平仓时间
	ClosedAt *time.Time `gorm:"column:closed_at;type:datetime" json:"closed_at"`
	// 状态
	Status string `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
}

// TableName 指定表名
func (PositionModel) TableName() string {
	return "positions"
}

// PositionRepositoryImpl 持仓仓储实现
type PositionRepositoryImpl struct {
	db *db.DB
}

// NewPositionRepository 创建持仓仓储
func NewPositionRepository(database *db.DB) domain.PositionRepository {
	return &PositionRepositoryImpl{
		db: database,
	}
}

// Save 保存持仓
func (pr *PositionRepositoryImpl) Save(ctx context.Context, position *domain.Position) error {
	model := &PositionModel{
		Model:         position.Model,
		PositionID:    position.PositionID,
		UserID:        position.UserID,
		Symbol:        position.Symbol,
		Side:          position.Side,
		Quantity:      position.Quantity.String(),
		EntryPrice:    position.EntryPrice.String(),
		CurrentPrice:  position.CurrentPrice.String(),
		UnrealizedPnL: position.UnrealizedPnL.String(),
		RealizedPnL:   position.RealizedPnL.String(),
		OpenedAt:      position.OpenedAt,
		ClosedAt:      position.ClosedAt,
		Status:        position.Status,
	}

	if err := pr.db.WithContext(ctx).Create(model).Error; err != nil {
		logger.Error(ctx, "Failed to save position",
			"position_id", position.PositionID,
			"error", err,
		)
		return fmt.Errorf("failed to save position: %w", err)
	}

	// 更新 domain 对象的 Model 信息
	position.Model = model.Model

	return nil
}

// Get 获取持仓
func (pr *PositionRepositoryImpl) Get(ctx context.Context, positionID string) (*domain.Position, error) {
	var model PositionModel

	if err := pr.db.WithContext(ctx).Where("position_id = ?", positionID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Error(ctx, "Failed to get position",
			"position_id", positionID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	return pr.modelToDomain(&model), nil
}

// GetByUser 获取用户持仓列表
func (pr *PositionRepositoryImpl) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Position, int64, error) {
	var models []PositionModel
	var total int64

	query := pr.db.WithContext(ctx).Where("user_id = ?", userID)

	if err := query.Model(&PositionModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count positions: %w", err)
	}

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&models).Error; err != nil {
		logger.Error(ctx, "Failed to get positions by user",
			"user_id", userID,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to get positions by user: %w", err)
	}

	positions := make([]*domain.Position, 0, len(models))
	for _, model := range models {
		positions = append(positions, pr.modelToDomain(&model))
	}

	return positions, total, nil
}

// GetBySymbol 获取交易对持仓列表
func (pr *PositionRepositoryImpl) GetBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*domain.Position, int64, error) {
	var models []PositionModel
	var total int64

	query := pr.db.WithContext(ctx).Where("symbol = ?", symbol)

	if err := query.Model(&PositionModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count positions: %w", err)
	}

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&models).Error; err != nil {
		logger.Error(ctx, "Failed to get positions by symbol",
			"symbol", symbol,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to get positions by symbol: %w", err)
	}

	positions := make([]*domain.Position, 0, len(models))
	for _, model := range models {
		positions = append(positions, pr.modelToDomain(&model))
	}

	return positions, total, nil
}

// Update 更新持仓
func (pr *PositionRepositoryImpl) Update(ctx context.Context, position *domain.Position) error {
	model := &PositionModel{
		Model:         position.Model,
		PositionID:    position.PositionID,
		UserID:        position.UserID,
		Symbol:        position.Symbol,
		Side:          position.Side,
		Quantity:      position.Quantity.String(),
		EntryPrice:    position.EntryPrice.String(),
		CurrentPrice:  position.CurrentPrice.String(),
		UnrealizedPnL: position.UnrealizedPnL.String(),
		RealizedPnL:   position.RealizedPnL.String(),
		OpenedAt:      position.OpenedAt,
		ClosedAt:      position.ClosedAt,
		Status:        position.Status,
	}

	if err := pr.db.WithContext(ctx).Model(&PositionModel{}).Where("position_id = ?", position.PositionID).Updates(model).Error; err != nil {
		logger.Error(ctx, "Failed to update position",
			"position_id", position.PositionID,
			"error", err,
		)
		return fmt.Errorf("failed to update position: %w", err)
	}

	return nil
}

// Close 平仓
func (pr *PositionRepositoryImpl) Close(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	if err := pr.db.WithContext(ctx).Model(&PositionModel{}).Where("position_id = ?", positionID).Updates(map[string]interface{}{
		"status":    "CLOSED",
		"closed_at": time.Now(),
	}).Error; err != nil {
		logger.Error(ctx, "Failed to close position",
			"position_id", positionID,
			"error", err,
		)
		return fmt.Errorf("failed to close position: %w", err)
	}

	return nil
}

// modelToDomain 将数据库模型转换为领域对象
func (pr *PositionRepositoryImpl) modelToDomain(model *PositionModel) *domain.Position {
	quantity, _ := decimal.NewFromString(model.Quantity)
	entryPrice, _ := decimal.NewFromString(model.EntryPrice)
	currentPrice, _ := decimal.NewFromString(model.CurrentPrice)
	unrealizedPnL, _ := decimal.NewFromString(model.UnrealizedPnL)
	realizedPnL, _ := decimal.NewFromString(model.RealizedPnL)

	return &domain.Position{
		Model:         model.Model,
		PositionID:    model.PositionID,
		UserID:        model.UserID,
		Symbol:        model.Symbol,
		Side:          model.Side,
		Quantity:      quantity,
		EntryPrice:    entryPrice,
		CurrentPrice:  currentPrice,
		UnrealizedPnL: unrealizedPnL,
		RealizedPnL:   realizedPnL,
		OpenedAt:      model.OpenedAt,
		ClosedAt:      model.ClosedAt,
		Status:        model.Status,
	}
}
