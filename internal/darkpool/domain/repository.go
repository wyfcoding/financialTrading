package domain

import (
	"context"
)

type DarkpoolRepository interface {
	SaveOrder(ctx context.Context, order *DarkOrder) error
	GetOrder(ctx context.Context, id string) (*DarkOrder, error)
	ListOrders(ctx context.Context, userID, status string) ([]*DarkOrder, error)
}
