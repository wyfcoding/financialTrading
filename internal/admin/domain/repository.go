package domain

import "context"

type AdminRepository interface {
	Save(ctx context.Context, admin *Admin) error
	GetByUsername(ctx context.Context, username string) (*Admin, error)
	GetByID(ctx context.Context, id uint) (*Admin, error)
}

type RoleRepository interface {
	Save(ctx context.Context, role *Role) error
	GetByName(ctx context.Context, name string) (*Role, error)
	GetByID(ctx context.Context, id uint) (*Role, error)
}
