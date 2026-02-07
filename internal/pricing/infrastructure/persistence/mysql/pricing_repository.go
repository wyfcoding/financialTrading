package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type pricingRepository struct {
	db *gorm.DB
}

// NewPricingRepository 创建并返回一个新的 pricingRepository 实例。
func NewPricingRepository(db *gorm.DB) domain.PricingRepository {
	return &pricingRepository{db: db}
}

// --- tx helpers ---

func (r *pricingRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *pricingRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *pricingRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *pricingRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

// --- Price (Simple Asset Price) ---

func (r *pricingRepository) SavePrice(ctx context.Context, price *domain.Price) error {
	model := toPriceModel(price)
	if model == nil {
		return nil
	}
	db := r.getDB(ctx).WithContext(ctx)
	if model.ID == 0 {
		if err := db.Create(model).Error; err != nil {
			return err
		}
		price.ID = model.ID
		price.CreatedAt = model.CreatedAt
		price.UpdatedAt = model.UpdatedAt
		return nil
	}
	return db.Model(&PriceModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"symbol":     model.Symbol,
			"bid":        model.Bid,
			"ask":        model.Ask,
			"mid":        model.Mid,
			"source":     model.Source,
			"timestamp":  model.Timestamp,
			"updated_at": time.Now(),
		}).Error
}

func (r *pricingRepository) GetLatestPrice(ctx context.Context, symbol string) (*domain.Price, error) {
	var model PriceModel
	err := r.getDB(ctx).WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("timestamp desc").
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toPrice(&model), nil
}

func (r *pricingRepository) ListLatestPrices(ctx context.Context, symbols []string) ([]*domain.Price, error) {
	prices := make([]*domain.Price, 0, len(symbols))
	for _, s := range symbols {
		p, err := r.GetLatestPrice(ctx, s)
		if err == nil && p != nil {
			prices = append(prices, p)
		}
	}
	return prices, nil
}

// --- PricingResult (Option/Derivatives Pricing) ---

func (r *pricingRepository) SavePricingResult(ctx context.Context, res *domain.PricingResult) error {
	model := toPricingResultModel(res)
	if model == nil {
		return nil
	}
	db := r.getDB(ctx).WithContext(ctx)
	if model.ID == 0 {
		if err := db.Create(model).Error; err != nil {
			return err
		}
		res.ID = model.ID
		res.CreatedAt = model.CreatedAt
		res.UpdatedAt = model.UpdatedAt
		return nil
	}
	return db.Model(&PricingResultModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"symbol":           model.Symbol,
			"option_price":     model.OptionPrice,
			"underlying_price": model.UnderlyingPrice,
			"delta":            model.Delta,
			"gamma":            model.Gamma,
			"theta":            model.Theta,
			"vega":             model.Vega,
			"rho":              model.Rho,
			"calculated_at":    model.CalculatedAt,
			"pricing_model":    model.PricingModel,
			"updated_at":       time.Now(),
		}).Error
}

func (r *pricingRepository) GetLatestPricingResult(ctx context.Context, symbol string) (*domain.PricingResult, error) {
	var m PricingResultModel
	if err := r.getDB(ctx).WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("calculated_at desc").
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toPricingResult(&m), nil
}

func (r *pricingRepository) GetPricingResultHistory(ctx context.Context, symbol string, limit int) ([]*domain.PricingResult, error) {
	var models []PricingResultModel
	if err := r.getDB(ctx).WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("calculated_at desc").
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.PricingResult, len(models))
	for i := range models {
		res[i] = toPricingResult(&models[i])
	}
	return res, nil
}

func (r *pricingRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}

// CleanupOldPrices maintenance function
func (r *pricingRepository) CleanupOldPrices(ctx context.Context, retention time.Duration) error {
	cutoff := time.Now().Add(-retention)
	return r.getDB(ctx).WithContext(ctx).Where("created_at < ?", cutoff).Delete(&PriceModel{}).Error
}
