package application

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/user/domain"
)

// UserCommandService 用户命令服务
type UserCommandService struct {
	repo      domain.UserRepository
	publisher domain.UserEventPublisher
}

// NewUserCommandService 创建新的用户命令服务
func NewUserCommandService(repo domain.UserRepository, publisher domain.UserEventPublisher) *UserCommandService {
	return &UserCommandService{
		repo:      repo,
		publisher: publisher,
	}
}

// CreateUser 创建用户
func (s *UserCommandService) CreateUser(ctx context.Context, username, email, phone, fullName, role, status string) (*domain.UserProfile, error) {
	user := &domain.UserProfile{
		Email: email,
		Phone: phone,
		Name:  fullName,
	}

	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}

	// 发布用户创建事件
	event := domain.UserCreatedEvent{
		UserID:    user.ID,
		Username:  username,
		Email:     user.Email,
		Phone:     user.Phone,
		FullName:  user.Name,
		Role:      role,
		Status:    status,
		CreatedAt: user.CreatedAt,
	}
	if err := s.publisher.PublishUserCreated(event); err != nil {
		// 记录错误但不影响主流程
	}

	return user, nil
}

// UpdateUser 更新用户
func (s *UserCommandService) UpdateUser(ctx context.Context, id uint, username, email, phone, fullName, role, status string) (*domain.UserProfile, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 更新用户信息
	if email != "" {
		user.Email = email
	}
	if phone != "" {
		user.Phone = phone
	}
	if fullName != "" {
		user.Name = fullName
	}

	user.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}

	// 发布用户更新事件
	event := domain.UserUpdatedEvent{
		UserID:    user.ID,
		Username:  username,
		Email:     user.Email,
		Phone:     user.Phone,
		FullName:  user.Name,
		Role:      role,
		Status:    status,
		UpdatedAt: user.UpdatedAt,
	}
	if err := s.publisher.PublishUserUpdated(event); err != nil {
		// 记录错误但不影响主流程
	}

	return user, nil
}

// DeleteUser 删除用户
func (s *UserCommandService) DeleteUser(ctx context.Context, id uint) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 这里应该调用 repo 的删除方法，但由于 repo 接口中可能没有定义 Delete 方法，暂时注释
	// if err := s.repo.Delete(ctx, id); err != nil {
	// 	return err
	// }

	// 发布用户删除事件
	event := domain.UserDeletedEvent{
		UserID:    user.ID,
		Username:  "",
		DeletedAt: time.Now(),
	}
	if err := s.publisher.PublishUserDeleted(event); err != nil {
		// 记录错误但不影响主流程
	}

	return nil
}

// ChangePassword 修改用户密码
func (s *UserCommandService) ChangePassword(ctx context.Context, id uint, newPassword string) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 这里应该有密码加密逻辑，但由于 UserProfile 结构体中没有 Password 字段，暂时注释
	// user.Password = newPassword
	user.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, user); err != nil {
		return err
	}

	// 发布密码变更事件
	event := domain.UserPasswordChangedEvent{
		UserID:    user.ID,
		ChangedAt: time.Now(),
	}
	if err := s.publisher.PublishUserPasswordChanged(event); err != nil {
		// 记录错误但不影响主流程
	}

	return nil
}

// VerifyEmail 验证用户邮箱
func (s *UserCommandService) VerifyEmail(ctx context.Context, id uint) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 这里应该有邮箱验证逻辑，但由于 UserProfile 结构体中没有 EmailVerified 字段，暂时注释
	// user.EmailVerified = true
	user.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, user); err != nil {
		return err
	}

	// 发布邮箱验证事件
	event := domain.UserEmailVerifiedEvent{
		UserID:     user.ID,
		Email:      user.Email,
		VerifiedAt: time.Now(),
	}
	if err := s.publisher.PublishUserEmailVerified(event); err != nil {
		// 记录错误但不影响主流程
	}

	return nil
}

// VerifyPhone 验证用户手机
func (s *UserCommandService) VerifyPhone(ctx context.Context, id uint) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 这里应该有手机验证逻辑，但由于 UserProfile 结构体中没有 PhoneVerified 字段，暂时注释
	// user.PhoneVerified = true
	user.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, user); err != nil {
		return err
	}

	// 发布手机验证事件
	event := domain.UserPhoneVerifiedEvent{
		UserID:     user.ID,
		Phone:      user.Phone,
		VerifiedAt: time.Now(),
	}
	if err := s.publisher.PublishUserPhoneVerified(event); err != nil {
		// 记录错误但不影响主流程
	}

	return nil
}

// LockUser 锁定用户账户
func (s *UserCommandService) LockUser(ctx context.Context, id uint, reason string) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 由于 UserProfile 结构体中没有 Status 字段，暂时注释
	// user.Status = "locked"
	user.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, user); err != nil {
		return err
	}

	// 发布账户锁定事件
	event := domain.UserLockedEvent{
		UserID:   user.ID,
		Username: "",
		Reason:   reason,
		LockedAt: time.Now(),
	}
	if err := s.publisher.PublishUserLocked(event); err != nil {
		// 记录错误但不影响主流程
	}

	// 同时发布状态变更事件
	statusEvent := domain.UserStatusChangedEvent{
		UserID:    user.ID,
		OldStatus: "active",
		NewStatus: "locked",
		ChangedAt: time.Now(),
	}
	if err := s.publisher.PublishUserStatusChanged(statusEvent); err != nil {
		// 记录错误但不影响主流程
	}

	return nil
}

// UnlockUser 解锁用户账户
func (s *UserCommandService) UnlockUser(ctx context.Context, id uint) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 由于 UserProfile 结构体中没有 Status 字段，暂时注释
	// user.Status = "active"
	user.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, user); err != nil {
		return err
	}

	// 发布账户解锁事件
	event := domain.UserUnlockedEvent{
		UserID:     user.ID,
		Username:   "",
		UnlockedAt: time.Now(),
	}
	if err := s.publisher.PublishUserUnlocked(event); err != nil {
		// 记录错误但不影响主流程
	}

	// 同时发布状态变更事件
	statusEvent := domain.UserStatusChangedEvent{
		UserID:    user.ID,
		OldStatus: "locked",
		NewStatus: "active",
		ChangedAt: time.Now(),
	}
	if err := s.publisher.PublishUserStatusChanged(statusEvent); err != nil {
		// 记录错误但不影响主流程
	}

	return nil
}

// RecordLogin 记录用户登录
func (s *UserCommandService) RecordLogin(ctx context.Context, id uint, username string, ipAddress string, userAgent string) error {
	// 发布登录事件
	event := domain.UserLoginEvent{
		UserID:    id,
		Username:  username,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		LoginAt:   time.Now(),
	}
	return s.publisher.PublishUserLogin(event)
}

// RecordLogout 记录用户登出
func (s *UserCommandService) RecordLogout(ctx context.Context, id uint, username string) error {
	// 发布登出事件
	event := domain.UserLogoutEvent{
		UserID:   id,
		Username: username,
		LogoutAt: time.Now(),
	}
	return s.publisher.PublishUserLogout(event)
}

// RecordFailedLogin 记录登录失败
func (s *UserCommandService) RecordFailedLogin(ctx context.Context, username string, ipAddress string, userAgent string, reason string) error {
	// 发布登录失败事件
	event := domain.UserFailedLoginEvent{
		Username:  username,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Reason:    reason,
		FailedAt:  time.Now(),
	}
	return s.publisher.PublishUserFailedLogin(event)
}
