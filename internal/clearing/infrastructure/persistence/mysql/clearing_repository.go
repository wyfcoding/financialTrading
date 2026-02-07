package mysql

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

// --- tx helpers ---

func (r *settlementRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *settlementRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *settlementRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *settlementRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *settlementRepository) Save(ctx context.Context, s *domain.Settlement) error {
	model := toSettlementModel(s)
	if model.ID == 0 {
		if err := r.getDB(ctx).WithContext(ctx).Create(model).Error; err != nil {
			return err
		}
		s.ID = model.ID
		s.CreatedAt = model.CreatedAt
		s.UpdatedAt = model.UpdatedAt
		return nil
	}

	return r.getDB(ctx).WithContext(ctx).
		Model(&SettlementModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"settlement_id": model.SettlementID,
			"trade_id":      model.TradeID,
			"buy_user_id":   model.BuyUserID,
			"sell_user_id":  model.SellUserID,
			"symbol":        model.Symbol,
			"quantity":      model.Quantity,
			"price":         model.Price,
			"total_amount":  model.TotalAmount,
			"fee":           model.Fee,
			"status":        model.Status,
			"settled_at":    model.SettledAt,
			"error_message": model.ErrorMessage,
		}).Error
}

func (r *settlementRepository) Get(ctx context.Context, id string) (*domain.Settlement, error) {
	var model SettlementModel
	if err := r.getDB(ctx).WithContext(ctx).Where("settlement_id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toSettlement(&model), nil
}

func (r *settlementRepository) GetByTradeID(ctx context.Context, tradeID string) (*domain.Settlement, error) {
	var model SettlementModel
	if err := r.getDB(ctx).WithContext(ctx).Where("trade_id = ?", tradeID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toSettlement(&model), nil
}

func (r *settlementRepository) List(ctx context.Context, limit int) ([]*domain.Settlement, error) {
	var models []*SettlementModel
	if err := r.getDB(ctx).WithContext(ctx).Limit(limit).Order("id desc").Find(&models).Error; err != nil {
		return nil, err
	}
	list := make([]*domain.Settlement, len(models))
	for i, model := range models {
		list[i] = toSettlement(model)
	}
	return list, nil
}

func (r *settlementRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
