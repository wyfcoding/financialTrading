package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/catalog/domain"
	"gorm.io/gorm"
)

type productRepository struct{ db *gorm.DB }

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Save(ctx context.Context, product *domain.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

func (r *productRepository) GetByID(ctx context.Context, id uint) (*domain.Product, error) {
	var p domain.Product
	err := r.db.WithContext(ctx).First(&p, id).Error
	return &p, err
}

func (r *productRepository) List(ctx context.Context, category string, offset, limit int) ([]*domain.Product, int, error) {
	var products []*domain.Product
	var total int64
	q := r.db.WithContext(ctx).Model(&domain.Product{})
	if category != "" {
		q = q.Where("category = ?", category)
	}
	q.Count(&total)
	err := q.Offset(offset).Limit(limit).Find(&products).Error
	return products, int(total), err
}
