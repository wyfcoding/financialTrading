package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/user/domain"
)

// UserService 用户应用服务，作为门面服务整合命令和查询服务
type UserService struct {
	commandService *UserCommandService
	queryService   *UserQueryService
}

// NewUserService 创建新的用户应用服务
func NewUserService(repo domain.UserRepository, publisher domain.UserEventPublisher) *UserService {
	commandService := NewUserCommandService(repo, publisher)
	queryService := NewUserQueryService(repo)

	return &UserService{
		commandService: commandService,
		queryService:   queryService,
	}
}

// GetUser 获取用户信息（查询操作）
func (s *UserService) GetUser(ctx context.Context, id uint) (*domain.UserProfile, error) {
	return s.queryService.GetUserByID(ctx, id)
}

// CreateUser 创建用户（命令操作）
func (s *UserService) CreateUser(ctx context.Context, username, email, phone, fullName, role, status string) (*domain.UserProfile, error) {
	return s.commandService.CreateUser(ctx, username, email, phone, fullName, role, status)
}

// UpdateUser 更新用户（命令操作）
func (s *UserService) UpdateUser(ctx context.Context, id uint, username, email, phone, fullName, role, status string) (*domain.UserProfile, error) {
	return s.commandService.UpdateUser(ctx, id, username, email, phone, fullName, role, status)
}

// DeleteUser 删除用户（命令操作）
func (s *UserService) DeleteUser(ctx context.Context, id uint) error {
	return s.commandService.DeleteUser(ctx, id)
}

// ChangePassword 修改用户密码（命令操作）
func (s *UserService) ChangePassword(ctx context.Context, id uint, newPassword string) error {
	return s.commandService.ChangePassword(ctx, id, newPassword)
}

// VerifyEmail 验证用户邮箱（命令操作）
func (s *UserService) VerifyEmail(ctx context.Context, id uint) error {
	return s.commandService.VerifyEmail(ctx, id)
}

// VerifyPhone 验证用户手机（命令操作）
func (s *UserService) VerifyPhone(ctx context.Context, id uint) error {
	return s.commandService.VerifyPhone(ctx, id)
}

// LockUser 锁定用户账户（命令操作）
func (s *UserService) LockUser(ctx context.Context, id uint, reason string) error {
	return s.commandService.LockUser(ctx, id, reason)
}

// UnlockUser 解锁用户账户（命令操作）
func (s *UserService) UnlockUser(ctx context.Context, id uint) error {
	return s.commandService.UnlockUser(ctx, id)
}

// RecordLogin 记录用户登录（命令操作）
func (s *UserService) RecordLogin(ctx context.Context, id uint, username string, ipAddress string, userAgent string) error {
	return s.commandService.RecordLogin(ctx, id, username, ipAddress, userAgent)
}

// RecordLogout 记录用户登出（命令操作）
func (s *UserService) RecordLogout(ctx context.Context, id uint, username string) error {
	return s.commandService.RecordLogout(ctx, id, username)
}

// RecordFailedLogin 记录登录失败（命令操作）
func (s *UserService) RecordFailedLogin(ctx context.Context, username string, ipAddress string, userAgent string, reason string) error {
	return s.commandService.RecordFailedLogin(ctx, username, ipAddress, userAgent, reason)
}

// ListUsers 列出所有用户（查询操作）
func (s *UserService) ListUsers(ctx context.Context, page, pageSize int) ([]*domain.UserProfile, int64, error) {
	return s.queryService.ListUsers(ctx, page, pageSize)
}

// GetUserByEmail 根据邮箱获取用户（查询操作）
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*domain.UserProfile, error) {
	return s.queryService.GetUserByEmail(ctx, email)
}

// GetUserByUsername 根据用户名获取用户（查询操作）
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*domain.UserProfile, error) {
	return s.queryService.GetUserByUsername(ctx, username)
}
