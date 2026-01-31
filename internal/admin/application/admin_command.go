package application

import (
	"context"
	"errors"
	"time"

	"github.com/wyfcoding/financialtrading/internal/admin/domain"
)

// LoginCommand 登录命令
type LoginCommand struct {
	Username string
	Password string
}

// CreateAdminCommand 创建管理员命令
type CreateAdminCommand struct {
	Username string
	Password string
	RoleID   uint
}

// CreateRoleCommand 创建角色命令
type CreateRoleCommand struct {
	Name        string
	Permissions string
}

// AdminCommandService 管理员命令服务
type AdminCommandService struct {
	adminRepo domain.AdminRepository
	roleRepo  domain.RoleRepository
	publisher domain.EventPublisher
}

// NewAdminCommandService 创建管理员命令服务实例
func NewAdminCommandService(
	adminRepo domain.AdminRepository,
	roleRepo domain.RoleRepository,
	publisher domain.EventPublisher,
) *AdminCommandService {
	return &AdminCommandService{
		adminRepo: adminRepo,
		roleRepo:  roleRepo,
		publisher: publisher,
	}
}

// Login 处理管理员登录
func (s *AdminCommandService) Login(ctx context.Context, cmd LoginCommand) (*AuthTokenDTO, error) {
	admin, err := s.adminRepo.GetByUsername(ctx, cmd.Username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Mock Password Check (In reality: bcrypt.CompareHashAndPassword)
	if admin.PasswordHash != cmd.Password { // Naive string compare for mock
		return nil, errors.New("invalid credentials")
	}

	// 发布登录事件
	event := domain.AdminLoggedInEvent{
		AdminID:   admin.ID,
		Username:  admin.Username,
		Timestamp: time.Now(),
	}
	s.publisher.Publish(ctx, "admin.login", cmd.Username, event)

	// Mock JWT Generation
	return &AuthTokenDTO{
		Token:     "mock_jwt_token_for_" + admin.Username,
		Type:      "Bearer",
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}, nil
}

// CreateAdmin 处理创建管理员
func (s *AdminCommandService) CreateAdmin(ctx context.Context, cmd CreateAdminCommand) (uint, error) {
	// Check if role exists
	_, err := s.roleRepo.GetByID(ctx, cmd.RoleID)
	if err != nil {
		return 0, errors.New("role not found")
	}

	// Hash password (mock)
	hashed := cmd.Password // bcrypt.GenerateFromPassword

	admin := domain.NewAdmin(cmd.Username, hashed, cmd.RoleID)
	if err := s.adminRepo.Save(ctx, admin); err != nil {
		return 0, err
	}

	// 发布创建事件
	event := domain.AdminCreatedEvent{
		AdminID:   admin.ID,
		Username:  admin.Username,
		RoleID:    admin.RoleID,
		Timestamp: time.Now(),
	}
	s.publisher.Publish(ctx, "admin.created", cmd.Username, event)

	return admin.ID, nil
}

// CreateRole 处理创建角色
func (s *AdminCommandService) CreateRole(ctx context.Context, cmd CreateRoleCommand) (uint, error) {
	role := domain.NewRole(cmd.Name, cmd.Permissions)
	if err := s.roleRepo.Save(ctx, role); err != nil {
		return 0, err
	}

	// 发布创建事件
	event := domain.RoleCreatedEvent{
		RoleID:    role.ID,
		Name:      role.Name,
		Timestamp: time.Now(),
	}
	s.publisher.Publish(ctx, "role.created", cmd.Name, event)

	return role.ID, nil
}
