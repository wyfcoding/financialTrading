// Package mysql 提供了持仓仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
	"github.com/wyfcoding/pkg/dtm"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PositionModel 持仓数据库模型，直接映射 positions 表。
// 遵循规范：嵌入 gorm.Model 并提供详细的业务字段注释。
type PositionModel struct {
	gorm.Model
	PositionID    string     `gorm:"column:position_id;type:varchar(32);uniqueIndex;not null;comment:持仓唯一标识"`
	UserID        string     `gorm:"column:user_id;type:varchar(32);index;not null;comment:所属用户ID"`
	Symbol        string     `gorm:"column:symbol;type:varchar(20);index;not null;comment:交易对名称"`
	Side          string     `gorm:"column:side;type:varchar(10);not null;comment:持仓方向(LONG/SHORT)"`
	Quantity      string     `gorm:"column:quantity;type:decimal(32,18);not null;comment:持仓数量"`
	EntryPrice    string     `gorm:"column:entry_price;type:decimal(32,18);not null;comment:开仓平均价格"`
	CurrentPrice  string     `gorm:"column:current_price;type:decimal(32,18);not null;comment:当前市场标记价格"`
	UnrealizedPnL string     `gorm:"column:unrealized_pnl;type:decimal(32,18);not null;comment:未实现盈亏"`
	RealizedPnL   string     `gorm:"column:realized_pnl;type:decimal(32,18);not null;comment:已实现盈亏"`
	OpenedAt      time.Time  `gorm:"column:opened_at;type:datetime;not null;comment:首次开仓时间"`
	ClosedAt      *time.Time `gorm:"column:closed_at;type:datetime;comment:最终平仓时间"`
	Status        string     `gorm:"column:status;type:varchar(20);index;not null;comment:持仓状态(OPEN/CLOSED)"`
	Version       int64      `gorm:"column:version;default:0;not null;comment:乐观锁版本号"`
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

// getDB 尝试从 Context 获取事务 DB，否则返回默认 DB
func (r *positionRepositoryImpl) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value("tx_db").(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

// Save 实现 domain.PositionRepository.Save
func (r *positionRepositoryImpl) Save(ctx context.Context, position *domain.Position) error {
	model := r.fromDomain(position)
	err := r.getDB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "position_id"}},
		UpdateAll: true,
	}).Create(model).Error
	if err != nil {
		logging.Error(ctx, "position_repository.save failed", "position_id", position.PositionID, "error", err)
		return fmt.Errorf("failed to save position: %w", err)
	}

	position.Model = model.Model
	return nil
}

// Get 实现 domain.PositionRepository.Get
func (r *positionRepositoryImpl) Get(ctx context.Context, positionID string) (*domain.Position, error) {
	var model PositionModel
	if err := r.getDB(ctx).Where("position_id = ?", positionID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "position_repository.get failed", "position_id", positionID, "error", err)
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	return r.toDomain(&model), nil
}

// GetByUser 实现 domain.PositionRepository.GetByUser
func (r *positionRepositoryImpl) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Position, int64, error) {
	var models []PositionModel
	var total int64
	db := r.getDB(ctx).Model(&PositionModel{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		logging.Error(ctx, "position_repository.get_by_user failed", "user_id", userID, "error", err)
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
	db := r.getDB(ctx).Model(&PositionModel{}).Where("symbol = ?", symbol)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		logging.Error(ctx, "position_repository.get_by_symbol failed", "symbol", symbol, "error", err)
		return nil, 0, fmt.Errorf("failed to get positions by symbol: %w", err)
	}

	positions := make([]*domain.Position, len(models))
	for i, m := range models {
		positions[i] = r.toDomain(&m)
	}
	return positions, total, nil
}

// Update 实现 domain.PositionRepository.Update (带乐观锁)
func (r *positionRepositoryImpl) Update(ctx context.Context, position *domain.Position) error {
	model := r.fromDomain(position)
	result := r.getDB(ctx).Model(&PositionModel{}).
		Where("position_id = ? AND version = ?", position.PositionID, position.Version).
		Updates(map[string]interface{}{
			"quantity":       model.Quantity,
			"entry_price":    model.EntryPrice,
			"current_price":  model.CurrentPrice,
			"unrealized_pnl": model.UnrealizedPnL,
			"realized_pnl":   model.RealizedPnL,
			"status":         model.Status,
			"version":        position.Version + 1, // 版本号递增
		})

	if result.Error != nil {
		logging.Error(ctx, "position_repository.update failed", "position_id", position.PositionID, "error", result.Error)
		return fmt.Errorf("failed to update position: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("concurrent update detected for position %s", position.PositionID)
	}

	position.Version++ // 更新本地版本号
	return nil
}

// Close 实现 domain.PositionRepository.Close
func (r *positionRepositoryImpl) Close(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	now := time.Now()
	err := r.getDB(ctx).Model(&PositionModel{}).Where("position_id = ?", positionID).Updates(map[string]interface{}{
		"status":        "CLOSED",
		"closed_at":     &now,
		"current_price": closePrice.String(),
		"version":       gorm.Expr("version + 1"),
	}).Error
	if err != nil {
		logging.Error(ctx, "position_repository.close failed", "position_id", positionID, "error", err)
		return fmt.Errorf("failed to close position: %w", err)
	}
	return nil
}

// ExecWithBarrier 在分布式事务屏障下执行业务逻辑
func (r *positionRepositoryImpl) ExecWithBarrier(ctx context.Context, barrier interface{}, fn func(ctx context.Context) error) error {
	return dtm.CallWithGorm(ctx, barrier, r.db, func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, "tx_db", tx)
		return fn(txCtx)
	})
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
		Version:       p.Version,
	}
}

func (r *positionRepositoryImpl) toDomain(m *PositionModel) *domain.Position {
	qty, err := decimal.NewFromString(m.Quantity)
	if err != nil {
		qty = decimal.Zero
	}
	entry, err := decimal.NewFromString(m.EntryPrice)
	if err != nil {
		entry = decimal.Zero
	}
	current, err := decimal.NewFromString(m.CurrentPrice)
	if err != nil {
		current = decimal.Zero
	}
	unrealized, err := decimal.NewFromString(m.UnrealizedPnL)
	if err != nil {
		unrealized = decimal.Zero
	}
	realized, err := decimal.NewFromString(m.RealizedPnL)
	if err != nil {
		realized = decimal.Zero
	}

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
		Version:       m.Version,
	}
}
