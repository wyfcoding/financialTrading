package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/catalog/domain"
)

type CatalogApplicationService struct{ repo domain.ProductRepository }

func NewCatalogApplicationService(repo domain.ProductRepository) *CatalogApplicationService {
	return &CatalogApplicationService{repo: repo}
}

func (s *CatalogApplicationService) CreateProduct(ctx context.Context, name, desc string, price float64, stock int, category string) (uint, error) {
	p := &domain.Product{Name: name, Description: desc, Price: price, Stock: stock, Category: category}
	if err := s.repo.Save(ctx, p); err != nil {
		return 0, err
	}
	return p.ID, nil
}

func (s *CatalogApplicationService) GetProduct(ctx context.Context, id uint) (*domain.Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CatalogApplicationService) ListProducts(ctx context.Context, category string, page, size int) ([]*domain.Product, int, error) {
	offset := (page - 1) * size
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, category, offset, size)
}
