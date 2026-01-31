package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/cart/domain"
	"gorm.io/gorm"
)

type cartRepository struct{ db *gorm.DB }

func NewCartRepository(db *gorm.DB) domain.CartRepository {
	return &cartRepository{db: db}
}

func (r *cartRepository) GetByUserID(ctx context.Context, userID string) (*domain.Cart, error) {
	var cart domain.Cart
	err := r.db.WithContext(ctx).Preload("Items").Where("user_id = ?", userID).First(&cart).Error
	return &cart, err
}

func (r *cartRepository) Save(ctx context.Context, cart *domain.Cart) error {
	return r.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(cart).Error
}

func (r *cartRepository) Delete(ctx context.Context, userID string) error {
	var cart domain.Cart
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&cart).Error; err != nil {
		return err
	}
	r.db.WithContext(ctx).Delete(&domain.CartItem{}, "cart_id = ?", cart.ID)
	return r.db.WithContext(ctx).Delete(&cart).Error
}
