package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
	"gorm.io/gorm"
)

// PricingResultModel 定价结果数据库模型
type PricingResultModel struct {
	gorm.Model
	Symbol          string `gorm:"column:symbol;type:varchar(32);index;not null" json:"symbol"`
	OptionPrice     string `gorm:"column:option_price;type:decimal(32,18);not null" json:"option_price"`
	UnderlyingPrice string `gorm:"column:underlying_price;type:decimal(32,18);not null" json:"underlying_price"`
	Delta           string `gorm:"column:delta;type:decimal(32,18)" json:"delta"`
	Gamma           string `gorm:"column:gamma;type:decimal(32,18)" json:"gamma"`
	Theta           string `gorm:"column:theta;type:decimal(32,18)" json:"theta"`
	Vega            string `gorm:"column:vega;type:decimal(32,18)" json:"vega"`
	Rho             string `gorm:"column:rho;type:decimal(32,18)" json:"rho"`
	CalculatedAt    int64  `gorm:"column:calculated_at;type:bigint;not null" json:"calculated_at"`
	PricingModel    string `gorm:"column:pricing_model;type:varchar(32)" json:"pricing_model"`
}

func (PricingResultModel) TableName() string { return "pricing_results" }

type pricingRepository struct {
	db *gorm.DB
}

// NewPricingRepository 创建并返回一个新的 pricingRepository 实例。
func NewPricingRepository(db *gorm.DB) domain.PricingRepository {
	return &pricingRepository{db: db}
}

// --- Price (Simple Asset Price) ---

func (r *pricingRepository) SavePrice(ctx context.Context, price *domain.Price) error {
	return r.db.WithContext(ctx).Create(price).Error
}

func (r *pricingRepository) GetLatestPrice(ctx context.Context, symbol string) (*domain.Price, error) {
	var price domain.Price
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp desc").First(&price).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &price, err
}

func (r *pricingRepository) ListLatestPrices(ctx context.Context, symbols []string) ([]*domain.Price, error) {
	var prices []*domain.Price
	// Fallback simple loop for NOW. Optimized implementation would use window functions or IN query with max ID logic.
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
	m := &PricingResultModel{
		PricingModel:    res.PricingModel,
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
	// Assuming Model field inside gorm.Model is handled automatically if we don't set ID
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *pricingRepository) GetLatestPricingResult(ctx context.Context, symbol string) (*domain.PricingResult, error) {
	var m PricingResultModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("calculated_at desc").First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

func (r *pricingRepository) GetPricingResultHistory(ctx context.Context, symbol string, limit int) ([]*domain.PricingResult, error) {
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

func (r *pricingRepository) toDomain(m *PricingResultModel) *domain.PricingResult {
	opPrice, _ := decimal.NewFromString(m.OptionPrice)
	ulPrice, _ := decimal.NewFromString(m.UnderlyingPrice)
	delta, _ := decimal.NewFromString(m.Delta)
	gamma, _ := decimal.NewFromString(m.Gamma)
	theta, _ := decimal.NewFromString(m.Theta)
	vega, _ := decimal.NewFromString(m.Vega)
	rho, _ := decimal.NewFromString(m.Rho)

	return &domain.PricingResult{
		PricingModel:    m.PricingModel,
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

// CleanupOldPrices maintenance function
func (r *pricingRepository) CleanupOldPrices(ctx context.Context, retention time.Duration) error {
	cutoff := time.Now().Add(-retention)
	return r.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&domain.Price{}).Error
}
