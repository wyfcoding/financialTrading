package application

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/catalog/domain"
)

// CreateProductCommand 创建商品命令
type CreateProductCommand struct {
	Name        string
	Description string
	Price       float64
	Stock       int
	Category    string
}

// UpdateProductCommand 更新商品命令
type UpdateProductCommand struct {
	ID          uint
	Name        string
	Description string
	Price       float64
	Stock       int
	Category    string
}

// CatalogCommandService 商品目录命令服务
type CatalogCommandService struct {
	repo      domain.ProductRepository
	publisher domain.EventPublisher
}

// NewCatalogCommandService 创建商品目录命令服务实例
func NewCatalogCommandService(
	repo domain.ProductRepository,
	publisher domain.EventPublisher,
) *CatalogCommandService {
	return &CatalogCommandService{
		repo:      repo,
		publisher: publisher,
	}
}

// CreateProduct 处理创建商品
func (s *CatalogCommandService) CreateProduct(ctx context.Context, cmd CreateProductCommand) (uint, error) {
	product := &domain.Product{
		Name:        cmd.Name,
		Description: cmd.Description,
		Price:       cmd.Price,
		Stock:       cmd.Stock,
		Category:    cmd.Category,
	}

	if err := s.repo.Save(ctx, product); err != nil {
		return 0, err
	}

	// 发布商品创建事件
	event := domain.ProductCreatedEvent{
		ProductID: product.ID,
		Name:      product.Name,
		Price:     product.Price,
		Stock:     product.Stock,
		Category:  product.Category,
		Timestamp: time.Now(),
	}
	s.publisher.Publish(ctx, "product.created", cmd.Name, event)

	return product.ID, nil
}

// UpdateProduct 处理更新商品
func (s *CatalogCommandService) UpdateProduct(ctx context.Context, cmd UpdateProductCommand) error {
	product, err := s.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return err
	}

	oldStock := product.Stock

	// 更新商品信息
	product.Name = cmd.Name
	product.Description = cmd.Description
	product.Price = cmd.Price
	product.Stock = cmd.Stock
	product.Category = cmd.Category

	if err := s.repo.Save(ctx, product); err != nil {
		return err
	}

	// 发布商品更新事件
	event := domain.ProductUpdatedEvent{
		ProductID: product.ID,
		Name:      product.Name,
		Price:     product.Price,
		Stock:     product.Stock,
		Category:  product.Category,
		Timestamp: time.Now(),
	}
	s.publisher.Publish(ctx, "product.updated", cmd.Name, event)

	// 如果库存发生变化，发布库存变更事件
	if oldStock != product.Stock {
		stockEvent := domain.ProductStockChangedEvent{
			ProductID: product.ID,
			OldStock:  oldStock,
			NewStock:  product.Stock,
			Timestamp: time.Now(),
		}
		s.publisher.Publish(ctx, "product.stock.changed", product.Name, stockEvent)
	}

	return nil
}
