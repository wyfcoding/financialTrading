package mysql

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
	"gorm.io/gorm"
)

type priceRepository struct {
	db *gorm.DB
}

func NewPriceRepository(db *gorm.DB) domain.PriceRepository {
	return &priceRepository{db: db}
}

func (r *priceRepository) Save(ctx context.Context, price *domain.Price) error {
	return r.db.WithContext(ctx).Create(price).Error
}

func (r *priceRepository) GetLatest(ctx context.Context, symbol string) (*domain.Price, error) {
	var price domain.Price
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp desc").First(&price).Error
	return &price, err
}

func (r *priceRepository) ListLatest(ctx context.Context, symbols []string) ([]*domain.Price, error) {
	var prices []*domain.Price
	// This is a naive implementation (N queries or complex subquery).
	// Ideally we SELECT distinct symbol or use window function.
	// For simplicity in this demo with MySQL 5.7/8.0 without window functions assumption:
	// We will just iterate. (Not optimize, but functional)

	// Better approach:
	// SELECT * FROM prices WHERE id IN (SELECT MAX(id) FROM prices GROUP BY symbol)
	// But let's assume valid time window for cache.

	// Optimized approach using window function (MySQL 8.0+)
	/*
		err := r.db.Raw(`
			SELECT * FROM (
				SELECT *, ROW_NUMBER() OVER (PARTITION BY symbol ORDER BY timestamp DESC) as rn
				FROM prices
				WHERE symbol IN ?
			) t WHERE t.rn = 1
		`, symbols).Scan(&prices).Error
	*/

	// Fallback simple loop for safety
	for _, s := range symbols {
		p, err := r.GetLatest(ctx, s)
		if err == nil {
			prices = append(prices, p)
		}
	}

	return prices, nil
}

// CleanupOldPrices is a maintenance function that could be run via a job
func (r *priceRepository) CleanupOldPrices(ctx context.Context, retention time.Duration) error {
	cutoff := time.Now().Add(-retention)
	return r.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&domain.Price{}).Error
}
