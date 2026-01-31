package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/catalog/domain"
)

// CatalogQueryService 商品目录查询服务
type CatalogQueryService struct {
	repo domain.ProductRepository
}

// NewCatalogQueryService 创建商品目录查询服务实例
func NewCatalogQueryService(
	repo domain.ProductRepository,
) *CatalogQueryService {
	return &CatalogQueryService{
		repo: repo,
	}
}

// GetProduct 根据ID获取商品信息
func (s *CatalogQueryService) GetProduct(ctx context.Context, id uint) (*domain.Product, error) {
	return s.repo.GetByID(ctx, id)
}

// ListProducts 列出商品
func (s *CatalogQueryService) ListProducts(ctx context.Context, category string, page, size int) ([]*domain.Product, int, error) {
	offset := (page - 1) * size
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, category, offset, size)
}
