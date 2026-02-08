package domain

import (
	"context"
)

type ContractRepository interface {
	Save(ctx context.Context, c *Contract) error
	Get(ctx context.Context, id string) (*Contract, error)
	List(ctx context.Context, underlying string, activeOnly bool) ([]Contract, error)
}
