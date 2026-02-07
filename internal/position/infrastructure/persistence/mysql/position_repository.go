package mysql

import (
	"context"
	"errors"
	"strconv"
	"time"

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

// --- tx helpers ---

func (r *positionRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *positionRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *positionRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *positionRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *positionRepository) Save(ctx context.Context, position *domain.Position) error {
	model := toPositionModel(position)
	if model == nil {
		return nil
	}

	db := r.getDB(ctx).WithContext(ctx)
	if model.ID == 0 {
		model.Lots = nil
		if err := db.Create(model).Error; err != nil {
			return err
		}
		position.ID = model.ID
		position.CreatedAt = model.CreatedAt
		position.UpdatedAt = model.UpdatedAt

		if len(position.Lots) > 0 {
			lots := toPositionLotModels(position.Lots)
			for i := range lots {
				lots[i].PositionID = model.ID
			}
			if err := db.Create(&lots).Error; err != nil {
				return err
			}
			position.Lots = toPositionLots(lots)
		}
		return nil
	}

	if err := db.Model(&PositionModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"user_id":             model.UserID,
			"symbol":              model.Symbol,
			"quantity":            model.Quantity,
			"average_entry_price": model.AverageEntryPrice,
			"realized_pnl":         model.RealizedPnL,
			"cost_method":         model.Method,
			"updated_at":          time.Now(),
		}).Error; err != nil {
		return err
	}

	// sync lots (simple strategy: delete + insert)
	if err := db.Where("position_id = ?", model.ID).Delete(&PositionLotModel{}).Error; err != nil {
		return err
	}
	if len(position.Lots) > 0 {
		lots := toPositionLotModels(position.Lots)
		for i := range lots {
			lots[i].PositionID = model.ID
		}
		if err := db.Create(&lots).Error; err != nil {
			return err
		}
		position.Lots = toPositionLots(lots)
	}
	position.UpdatedAt = time.Now()
	return nil
}

func (r *positionRepository) GetByUserSymbol(ctx context.Context, userID, symbol string) (*domain.Position, error) {
	var model PositionModel
	err := r.getDB(ctx).WithContext(ctx).
		Preload("Lots").
		Where("user_id = ? AND symbol = ?", userID, symbol).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toPosition(&model), nil
}

func (r *positionRepository) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Position, int64, error) {
	var models []PositionModel
	var total int64
	query := r.getDB(ctx).WithContext(ctx).Model(&PositionModel{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Limit(limit).Offset(offset).Order("updated_at desc").Find(&models).Error; err != nil {
		return nil, 0, err
	}
	positions := make([]*domain.Position, len(models))
	for i := range models {
		positions[i] = toPosition(&models[i])
	}
	return positions, total, nil
}

func (r *positionRepository) GetBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*domain.Position, int64, error) {
	var models []PositionModel
	var total int64
	query := r.getDB(ctx).WithContext(ctx).Model(&PositionModel{}).Where("symbol = ?", symbol)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Limit(limit).Offset(offset).Order("updated_at desc").Find(&models).Error; err != nil {
		return nil, 0, err
	}
	positions := make([]*domain.Position, len(models))
	for i := range models {
		positions[i] = toPosition(&models[i])
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

func (r *positionRepository) Get(ctx context.Context, positionID string) (*domain.Position, error) {
	id, err := parsePositionID(positionID)
	if err != nil {
		return nil, err
	}
	var model PositionModel
	err = r.getDB(ctx).WithContext(ctx).Preload("Lots").First(&model, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toPosition(&model), nil
}

func (r *positionRepository) Update(ctx context.Context, position *domain.Position) error {
	return r.Save(ctx, position)
}

func (r *positionRepository) Close(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	id, err := parsePositionID(positionID)
	if err != nil {
		return err
	}
	return r.getDB(ctx).WithContext(ctx).
		Model(&PositionModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"quantity":            0,
			"average_entry_price": 0,
			"updated_at":          time.Now(),
		}).Error
}

func parsePositionID(positionID string) (uint, error) {
	if positionID == "" {
		return 0, errors.New("position_id is required")
	}
	parsed, err := strconv.ParseUint(positionID, 10, 64)
	if err != nil {
		return 0, errors.New("invalid position_id")
	}
	return uint(parsed), nil
}
