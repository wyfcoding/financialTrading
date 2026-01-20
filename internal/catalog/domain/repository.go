package domain

import "context"

type ProductRepository interface {
	Save(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id uint) (*Product, error)
	List(ctx context.Context, category string, offset, limit int) ([]*Product, int, error)
}
