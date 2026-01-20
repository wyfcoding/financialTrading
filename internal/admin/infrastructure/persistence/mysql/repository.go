package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/admin/domain"
	"gorm.io/gorm"
)

// Admin Repository
type adminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) domain.AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) Save(ctx context.Context, admin *domain.Admin) error {
	return r.db.WithContext(ctx).Save(admin).Error
}

func (r *adminRepository) GetByUsername(ctx context.Context, username string) (*domain.Admin, error) {
	var admin domain.Admin
	err := r.db.WithContext(ctx).Where("username = ?", username).Preload("Role").First(&admin).Error
	return &admin, err
}

func (r *adminRepository) GetByID(ctx context.Context, id uint) (*domain.Admin, error) {
	var admin domain.Admin
	err := r.db.WithContext(ctx).Preload("Role").First(&admin, id).Error
	return &admin, err
}

// Role Repository
type roleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) domain.RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) Save(ctx context.Context, role *domain.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *roleRepository) GetByName(ctx context.Context, name string) (*domain.Role, error) {
	var role domain.Role
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error
	return &role, err
}

func (r *roleRepository) GetByID(ctx context.Context, id uint) (*domain.Role, error) {
	var role domain.Role
	err := r.db.WithContext(ctx).First(&role, id).Error
	return &role, err
}
