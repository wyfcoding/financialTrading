package domain

import "context"

type UserRepository interface {
	Save(ctx context.Context, user *UserProfile) error
	GetByID(ctx context.Context, id uint) (*UserProfile, error)
	GetByEmail(ctx context.Context, email string) (*UserProfile, error)
}
