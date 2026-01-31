package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/admin/domain"
)

// AdminQueryService 管理员查询服务
type AdminQueryService struct {
	adminRepo domain.AdminRepository
	roleRepo  domain.RoleRepository
}

// NewAdminQueryService 创建管理员查询服务实例
func NewAdminQueryService(
	adminRepo domain.AdminRepository,
	roleRepo domain.RoleRepository,
) *AdminQueryService {
	return &AdminQueryService{
		adminRepo: adminRepo,
		roleRepo:  roleRepo,
	}
}

// GetAdmin 根据ID获取管理员信息
func (s *AdminQueryService) GetAdmin(ctx context.Context, id uint) (*AdminDTO, error) {
	admin, err := s.adminRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &AdminDTO{
		ID:        admin.ID,
		Username:  admin.Username,
		RoleName:  admin.Role.Name,
		CreatedAt: admin.CreatedAt.Unix(),
	}, nil
}

// GetAdminByUsername 根据用户名获取管理员信息
func (s *AdminQueryService) GetAdminByUsername(ctx context.Context, username string) (*AdminDTO, error) {
	admin, err := s.adminRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return &AdminDTO{
		ID:        admin.ID,
		Username:  admin.Username,
		RoleName:  admin.Role.Name,
		CreatedAt: admin.CreatedAt.Unix(),
	}, nil
}

// GetRole 根据ID获取角色信息
func (s *AdminQueryService) GetRole(ctx context.Context, id uint) (*RoleDTO, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &RoleDTO{
		ID:          role.ID,
		Name:        role.Name,
		Permissions: role.Permissions,
		CreatedAt:   role.CreatedAt.Unix(),
	}, nil
}

// GetRoleByName 根据名称获取角色信息
func (s *AdminQueryService) GetRoleByName(ctx context.Context, name string) (*RoleDTO, error) {
	role, err := s.roleRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	return &RoleDTO{
		ID:          role.ID,
		Name:        role.Name,
		Permissions: role.Permissions,
		CreatedAt:   role.CreatedAt.Unix(),
	}, nil
}
