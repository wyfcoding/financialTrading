package infrastructure

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/darkpool/domain"
	"gorm.io/gorm"
)

type GormDarkpoolRepository struct {
	db *gorm.DB
}

func NewGormDarkpoolRepository(db *gorm.DB) *GormDarkpoolRepository {
	return &GormDarkpoolRepository{db: db}
}

func (r *GormDarkpoolRepository) SaveOrder(ctx context.Context, order *domain.DarkOrder) error {
	return r.db.WithContext(ctx).Save(order).Error
}

func (r *GormDarkpoolRepository) GetOrder(ctx context.Context, id string) (*domain.DarkOrder, error) {
	var order domain.DarkOrder
	err := r.db.WithContext(ctx).Where("order_id = ?", id).First(&order).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &order, err
}

func (r *GormDarkpoolRepository) ListOrders(ctx context.Context, userID, status string) ([]*domain.DarkOrder, error) {
	var orders []*domain.DarkOrder
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&orders).Error
	return orders, err
}
