package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/cart/domain"
	"gorm.io/gorm"
)

// CartQueryService 购物车查询服务
type CartQueryService struct {
	repo domain.CartRepository
}

// NewCartQueryService 创建购物车查询服务实例
func NewCartQueryService(
	repo domain.CartRepository,
) *CartQueryService {
	return &CartQueryService{
		repo: repo,
	}
}

// GetCart 根据用户ID获取购物车信息
func (s *CartQueryService) GetCart(ctx context.Context, userID string) (*domain.Cart, error) {
	cart, err := s.repo.GetByUserID(ctx, userID)
	if err == gorm.ErrRecordNotFound {
		return &domain.Cart{UserID: userID}, nil
	}
	return cart, err
}

// GetCartTotal 获取购物车总金额
func (s *CartQueryService) GetCartTotal(ctx context.Context, userID string) (float64, error) {
	cart, err := s.repo.GetByUserID(ctx, userID)
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return cart.Total(), nil
}

// GetCartItemCount 获取购物车商品数量
func (s *CartQueryService) GetCartItemCount(ctx context.Context, userID string) (int, error) {
	cart, err := s.repo.GetByUserID(ctx, userID)
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return len(cart.Items), nil
}
