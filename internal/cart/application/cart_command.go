package application

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/cart/domain"
	"gorm.io/gorm"
)

// AddItemCommand 添加商品到购物车命令
type AddItemCommand struct {
	UserID    string
	ProductID string
	Quantity  int
	Price     float64
}

// RemoveItemCommand 从购物车移除商品命令
type RemoveItemCommand struct {
	UserID    string
	ProductID string
}

// ClearCartCommand 清空购物车命令
type ClearCartCommand struct {
	UserID string
}

// CartCommandService 购物车命令服务
type CartCommandService struct {
	repo      domain.CartRepository
	publisher domain.EventPublisher
}

// NewCartCommandService 创建购物车命令服务实例
func NewCartCommandService(
	repo domain.CartRepository,
	publisher domain.EventPublisher,
) *CartCommandService {
	return &CartCommandService{
		repo:      repo,
		publisher: publisher,
	}
}

// AddItem 处理添加商品到购物车
func (s *CartCommandService) AddItem(ctx context.Context, cmd AddItemCommand) error {
	cart, err := s.repo.GetByUserID(ctx, cmd.UserID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	if cart.ID == 0 {
		cart = &domain.Cart{UserID: cmd.UserID}
		if err := s.repo.Save(ctx, cart); err != nil {
			return err
		}

		// 发布购物车创建事件
		event := domain.CartCreatedEvent{
			CartID:    cart.ID,
			UserID:    cart.UserID,
			Timestamp: time.Now(),
		}
		s.publisher.Publish(ctx, "cart.created", cmd.UserID, event)
	}

	cart.AddItem(cmd.ProductID, cmd.Quantity, cmd.Price)
	if err := s.repo.Save(ctx, cart); err != nil {
		return err
	}

	// 发布添加商品事件
	event := domain.CartItemAddedEvent{
		CartID:    cart.ID,
		UserID:    cart.UserID,
		ProductID: cmd.ProductID,
		Quantity:  cmd.Quantity,
		Price:     cmd.Price,
		Timestamp: time.Now(),
	}
	s.publisher.Publish(ctx, "cart.item.added", cmd.UserID, event)

	return nil
}

// RemoveItem 处理从购物车移除商品
func (s *CartCommandService) RemoveItem(ctx context.Context, cmd RemoveItemCommand) error {
	cart, err := s.repo.GetByUserID(ctx, cmd.UserID)
	if err != nil {
		return err
	}

	cart.RemoveItem(cmd.ProductID)
	if err := s.repo.Save(ctx, cart); err != nil {
		return err
	}

	// 发布移除商品事件
	event := domain.CartItemRemovedEvent{
		CartID:    cart.ID,
		UserID:    cart.UserID,
		ProductID: cmd.ProductID,
		Timestamp: time.Now(),
	}
	s.publisher.Publish(ctx, "cart.item.removed", cmd.UserID, event)

	return nil
}

// ClearCart 处理清空购物车
func (s *CartCommandService) ClearCart(ctx context.Context, cmd ClearCartCommand) error {
	cart, err := s.repo.GetByUserID(ctx, cmd.UserID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	if err := s.repo.Delete(ctx, cmd.UserID); err != nil {
		return err
	}

	// 发布清空购物车事件
	if cart.ID != 0 {
		event := domain.CartClearedEvent{
			CartID:    cart.ID,
			UserID:    cart.UserID,
			Timestamp: time.Now(),
		}
		s.publisher.Publish(ctx, "cart.cleared", cmd.UserID, event)
	}

	return nil
}
