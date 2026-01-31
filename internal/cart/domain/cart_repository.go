package domain

import "context"

type CartRepository interface {
	GetByUserID(ctx context.Context, userID string) (*Cart, error)
	Save(ctx context.Context, cart *Cart) error
	Delete(ctx context.Context, userID string) error
}
