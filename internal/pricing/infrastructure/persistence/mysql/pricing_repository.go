// Package mysql 提供了定价结果仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
	"gorm.io/gorm"
)

// PricingResultModel 定价结果数据库模型
type PricingResultModel struct {
	gorm.Model
	Symbol          string `gorm:"column:symbol;type:varchar(32);index;not null"`
	OptionPrice     string `gorm:"column:option_price;type:decimal(32,18);not null"`
	UnderlyingPrice string `gorm:"column:underlying_price;type:decimal(32,18);not null"`
	Delta           string `gorm:"column:delta;type:decimal(32,18)"`
	Gamma           string `gorm:"column:gamma;type:decimal(32,18)"`
	Theta           string `gorm:"column:theta;type:decimal(32,18)"`
	Vega            string `gorm:"column:vega;type:decimal(32,18)"`
	Rho             string `gorm:"column:rho;type:decimal(32,18)"`
	CalculatedAt    int64  `gorm:"column:calculated_at;type:bigint;not null"`
}

func (PricingResultModel) TableName() string { return "pricing_results" }

type pricingRepositoryImpl struct {
	db *gorm.DB
}

func NewPricingRepository(db *gorm.DB) domain.PricingRepository {
	return &pricingRepositoryImpl{db: db}
}

func (r *pricingRepositoryImpl) Save(ctx context.Context, res *domain.PricingResult) error {
	m := &PricingResultModel{
		Model:           res.Model,
		Symbol:          res.Symbol,
		OptionPrice:     res.OptionPrice.String(),
		UnderlyingPrice: res.UnderlyingPrice.String(),
		Delta:           res.Delta.String(),
		Gamma:           res.Gamma.String(),
		Theta:           res.Theta.String(),
		Vega:            res.Vega.String(),
		Rho:             res.Rho.String(),
		CalculatedAt:    res.CalculatedAt,
	}
	err := r.db.WithContext(ctx).Create(m).Error
	if err == nil {
		res.Model = m.Model
	}
	return err
}

func (r *pricingRepositoryImpl) GetLatest(ctx context.Context, symbol string) (*domain.PricingResult, error) {
	var m PricingResultModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("calculated_at desc").First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

func (r *pricingRepositoryImpl) GetHistory(ctx context.Context, symbol string, limit int) ([]*domain.PricingResult, error) {
	var models []PricingResultModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("calculated_at desc").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.PricingResult, len(models))
	for i, m := range models {
		res[i] = r.toDomain(&m)
	}
	return res, nil
}

func (r *pricingRepositoryImpl) toDomain(m *PricingResultModel) *domain.PricingResult {
	opPrice, err := decimal.NewFromString(m.OptionPrice)
	if err != nil {
		opPrice = decimal.Zero
	}
	ulPrice, err := decimal.NewFromString(m.UnderlyingPrice)
	if err != nil {
		ulPrice = decimal.Zero
	}
	delta, err := decimal.NewFromString(m.Delta)
	if err != nil {
		delta = decimal.Zero
	}
	gamma, err := decimal.NewFromString(m.Gamma)
	if err != nil {
		gamma = decimal.Zero
	}
	theta, err := decimal.NewFromString(m.Theta)
	if err != nil {
		theta = decimal.Zero
	}
	vega, err := decimal.NewFromString(m.Vega)
	if err != nil {
		vega = decimal.Zero
	}
	rho, err := decimal.NewFromString(m.Rho)
	if err != nil {
		rho = decimal.Zero
	}

	return &domain.PricingResult{
		Model:           m.Model,
		Symbol:          m.Symbol,
		OptionPrice:     opPrice,
		UnderlyingPrice: ulPrice,
		Delta:           delta,
		Gamma:           gamma,
		Theta:           theta,
		Vega:            vega,
		Rho:             rho,
		CalculatedAt:    m.CalculatedAt,
	}
}
