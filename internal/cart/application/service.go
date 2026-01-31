package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/cart/domain"
)

// CartApplicationService 购物车服务门面，整合命令服务和查询服务
type CartApplicationService struct {
	commandService *CartCommandService
	queryService   *CartQueryService
}

// NewCartApplicationService 创建购物车服务门面实例
func NewCartApplicationService(
	repo domain.CartRepository,
	publisher domain.EventPublisher,
) *CartApplicationService {
	return &CartApplicationService{
		commandService: NewCartCommandService(repo, publisher),
		queryService:   NewCartQueryService(repo),
	}
}

// GetCart 根据用户ID获取购物车信息
func (s *CartApplicationService) GetCart(ctx context.Context, userID string) (*domain.Cart, error) {
	return s.queryService.GetCart(ctx, userID)
}

// GetCartTotal 获取购物车总金额
func (s *CartApplicationService) GetCartTotal(ctx context.Context, userID string) (float64, error) {
	return s.queryService.GetCartTotal(ctx, userID)
}

// GetCartItemCount 获取购物车商品数量
func (s *CartApplicationService) GetCartItemCount(ctx context.Context, userID string) (int, error) {
	return s.queryService.GetCartItemCount(ctx, userID)
}

// AddItem 处理添加商品到购物车
func (s *CartApplicationService) AddItem(ctx context.Context, userID, productID string, qty int, price float64) error {
	cmd := AddItemCommand{
		UserID:    userID,
		ProductID: productID,
		Quantity:  qty,
		Price:     price,
	}
	return s.commandService.AddItem(ctx, cmd)
}

// RemoveItem 处理从购物车移除商品
func (s *CartApplicationService) RemoveItem(ctx context.Context, userID, productID string) error {
	cmd := RemoveItemCommand{
		UserID:    userID,
		ProductID: productID,
	}
	return s.commandService.RemoveItem(ctx, cmd)
}

// ClearCart 处理清空购物车
func (s *CartApplicationService) ClearCart(ctx context.Context, userID string) error {
	cmd := ClearCartCommand{
		UserID: userID,
	}
	return s.commandService.ClearCart(ctx, cmd)
}
