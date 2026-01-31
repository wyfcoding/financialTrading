package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/admin/domain"
)

// AdminApplicationService 管理员服务门面，整合命令服务和查询服务
type AdminApplicationService struct {
	commandService *AdminCommandService
	queryService   *AdminQueryService
}

// NewAdminApplicationService 创建管理员服务门面实例
func NewAdminApplicationService(
	adminRepo domain.AdminRepository,
	roleRepo domain.RoleRepository,
	publisher domain.EventPublisher,
) *AdminApplicationService {
	return &AdminApplicationService{
		commandService: NewAdminCommandService(adminRepo, roleRepo, publisher),
		queryService:   NewAdminQueryService(adminRepo, roleRepo),
	}
}

// Login 处理管理员登录
func (s *AdminApplicationService) Login(ctx context.Context, cmd LoginCommand) (*AuthTokenDTO, error) {
	return s.commandService.Login(ctx, cmd)
}

// CreateAdmin 处理创建管理员
func (s *AdminApplicationService) CreateAdmin(ctx context.Context, cmd CreateAdminCommand) (uint, error) {
	return s.commandService.CreateAdmin(ctx, cmd)
}

// CreateRole 处理创建角色
func (s *AdminApplicationService) CreateRole(ctx context.Context, cmd CreateRoleCommand) (uint, error) {
	return s.commandService.CreateRole(ctx, cmd)
}

// GetAdmin 根据ID获取管理员信息
func (s *AdminApplicationService) GetAdmin(ctx context.Context, id uint) (*AdminDTO, error) {
	return s.queryService.GetAdmin(ctx, id)
}

// GetAdminByUsername 根据用户名获取管理员信息
func (s *AdminApplicationService) GetAdminByUsername(ctx context.Context, username string) (*AdminDTO, error) {
	return s.queryService.GetAdminByUsername(ctx, username)
}

// GetRole 根据ID获取角色信息
func (s *AdminApplicationService) GetRole(ctx context.Context, id uint) (*RoleDTO, error) {
	return s.queryService.GetRole(ctx, id)
}

// GetRoleByName 根据名称获取角色信息
func (s *AdminApplicationService) GetRoleByName(ctx context.Context, name string) (*RoleDTO, error) {
	return s.queryService.GetRoleByName(ctx, name)
}
