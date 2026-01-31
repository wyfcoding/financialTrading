package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/admin/domain"
)

// AdminService 管理员服务门面，整合命令服务和查询服务
type AdminService struct {
	commandService *AdminCommandService
	queryService   *AdminQueryService
}

// NewAdminService 创建管理员服务门面实例
func NewAdminService(
	adminRepo domain.AdminRepository,
	roleRepo domain.RoleRepository,
	publisher domain.EventPublisher,
) *AdminService {
	return &AdminService{
		commandService: NewAdminCommandService(adminRepo, roleRepo, publisher),
		queryService:   NewAdminQueryService(adminRepo, roleRepo),
	}
}

// Login 处理管理员登录
func (s *AdminService) Login(ctx context.Context, cmd LoginCommand) (*AuthTokenDTO, error) {
	return s.commandService.Login(ctx, cmd)
}

// CreateAdmin 处理创建管理员
func (s *AdminService) CreateAdmin(ctx context.Context, cmd CreateAdminCommand) (uint, error) {
	return s.commandService.CreateAdmin(ctx, cmd)
}

// CreateRole 处理创建角色
func (s *AdminService) CreateRole(ctx context.Context, cmd CreateRoleCommand) (uint, error) {
	return s.commandService.CreateRole(ctx, cmd)
}

// GetAdmin 根据ID获取管理员信息
func (s *AdminService) GetAdmin(ctx context.Context, id uint) (*AdminDTO, error) {
	return s.queryService.GetAdmin(ctx, id)
}

// GetAdminByUsername 根据用户名获取管理员信息
func (s *AdminService) GetAdminByUsername(ctx context.Context, username string) (*AdminDTO, error) {
	return s.queryService.GetAdminByUsername(ctx, username)
}

// GetRole 根据ID获取角色信息
func (s *AdminService) GetRole(ctx context.Context, id uint) (*RoleDTO, error) {
	return s.queryService.GetRole(ctx, id)
}

// GetRoleByName 根据名称获取角色信息
func (s *AdminService) GetRoleByName(ctx context.Context, name string) (*RoleDTO, error) {
	return s.queryService.GetRoleByName(ctx, name)
}

// --- DTOs ---

type AuthTokenDTO struct {
	Token     string
	Type      string
	ExpiresAt int64
}

type AdminDTO struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	RoleName  string `json:"role_name"`
	CreatedAt int64  `json:"created_at"`
}

type RoleDTO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Permissions string `json:"permissions"`
	CreatedAt   int64  `json:"created_at"`
}
