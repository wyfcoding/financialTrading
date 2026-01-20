package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/cart/domain"
	"gorm.io/gorm"
)

type CartApplicationService struct{ repo domain.CartRepository }

func NewCartApplicationService(repo domain.CartRepository) *CartApplicationService {
	return &CartApplicationService{repo: repo}
}

func (s *CartApplicationService) GetCart(ctx context.Context, userID string) (*domain.Cart, error) {
	cart, err := s.repo.GetByUserID(ctx, userID)
	if err == gorm.ErrRecordNotFound {
		return &domain.Cart{UserID: userID}, nil
	}
	return cart, err
}

func (s *CartApplicationService) AddItem(ctx context.Context, userID, productID string, qty int, price float64) error {
	cart, _ := s.repo.GetByUserID(ctx, userID)
	if cart.ID == 0 {
		cart = &domain.Cart{UserID: userID}
	}
	cart.AddItem(productID, qty, price)
	return s.repo.Save(ctx, cart)
}

func (s *CartApplicationService) RemoveItem(ctx context.Context, userID, productID string) error {
	cart, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	cart.RemoveItem(productID)
	return s.repo.Save(ctx, cart)
}

func (s *CartApplicationService) ClearCart(ctx context.Context, userID string) error {
	return s.repo.Delete(ctx, userID)
}
