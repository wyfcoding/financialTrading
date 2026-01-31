package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/catalog/domain"
)

// CatalogApplicationService 商品目录服务门面，整合命令服务和查询服务
type CatalogApplicationService struct {
	commandService *CatalogCommandService
	queryService   *CatalogQueryService
}

// NewCatalogApplicationService 创建商品目录服务门面实例
func NewCatalogApplicationService(
	repo domain.ProductRepository,
	publisher domain.EventPublisher,
) *CatalogApplicationService {
	return &CatalogApplicationService{
		commandService: NewCatalogCommandService(repo, publisher),
		queryService:   NewCatalogQueryService(repo),
	}
}

// CreateProduct 处理创建商品
func (s *CatalogApplicationService) CreateProduct(ctx context.Context, name, desc string, price float64, stock int, category string) (uint, error) {
	cmd := CreateProductCommand{
		Name:        name,
		Description: desc,
		Price:       price,
		Stock:       stock,
		Category:    category,
	}
	return s.commandService.CreateProduct(ctx, cmd)
}

// UpdateProduct 处理更新商品
func (s *CatalogApplicationService) UpdateProduct(ctx context.Context, id uint, name, desc string, price float64, stock int, category string) error {
	cmd := UpdateProductCommand{
		ID:          id,
		Name:        name,
		Description: desc,
		Price:       price,
		Stock:       stock,
		Category:    category,
	}
	return s.commandService.UpdateProduct(ctx, cmd)
}

// GetProduct 根据ID获取商品信息
func (s *CatalogApplicationService) GetProduct(ctx context.Context, id uint) (*domain.Product, error) {
	return s.queryService.GetProduct(ctx, id)
}

// ListProducts 列出商品
func (s *CatalogApplicationService) ListProducts(ctx context.Context, category string, page, size int) ([]*domain.Product, int, error) {
	return s.queryService.ListProducts(ctx, category, page, size)
}
