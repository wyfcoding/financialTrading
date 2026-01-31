package domain

// UserEventPublisher 用户事件发布者接口
type UserEventPublisher interface {
	// PublishUserCreated 发布用户创建事件
	PublishUserCreated(event UserCreatedEvent) error

	// PublishUserUpdated 发布用户更新事件
	PublishUserUpdated(event UserUpdatedEvent) error

	// PublishUserDeleted 发布用户删除事件
	PublishUserDeleted(event UserDeletedEvent) error

	// PublishUserStatusChanged 发布用户状态变更事件
	PublishUserStatusChanged(event UserStatusChangedEvent) error

	// PublishUserRoleChanged 发布用户角色变更事件
	PublishUserRoleChanged(event UserRoleChangedEvent) error

	// PublishUserPasswordChanged 发布用户密码变更事件
	PublishUserPasswordChanged(event UserPasswordChangedEvent) error

	// PublishUserEmailVerified 发布用户邮箱验证事件
	PublishUserEmailVerified(event UserEmailVerifiedEvent) error

	// PublishUserPhoneVerified 发布用户手机验证事件
	PublishUserPhoneVerified(event UserPhoneVerifiedEvent) error

	// PublishUserLogin 发布用户登录事件
	PublishUserLogin(event UserLoginEvent) error

	// PublishUserLogout 发布用户登出事件
	PublishUserLogout(event UserLogoutEvent) error

	// PublishUserFailedLogin 发布用户登录失败事件
	PublishUserFailedLogin(event UserFailedLoginEvent) error

	// PublishUserLocked 发布用户账户锁定事件
	PublishUserLocked(event UserLockedEvent) error

	// PublishUserUnlocked 发布用户账户解锁事件
	PublishUserUnlocked(event UserUnlockedEvent) error
}
