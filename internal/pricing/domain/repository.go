package domain

import "context"

type PriceRepository interface {
	Save(ctx context.Context, price *Price) error
	GetLatest(ctx context.Context, symbol string) (*Price, error)
	ListLatest(ctx context.Context, symbols []string) ([]*Price, error)
}
