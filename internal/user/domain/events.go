package domain

import (
	"time"
)

// UserCreatedEvent 用户创建事件
type UserCreatedEvent struct {
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone,omitempty"`
	FullName  string    `json:"full_name,omitempty"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// UserUpdatedEvent 用户更新事件
type UserUpdatedEvent struct {
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	Email     string    `json:"email,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	FullName  string    `json:"full_name,omitempty"`
	Role      string    `json:"role,omitempty"`
	Status    string    `json:"status,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserDeletedEvent 用户删除事件
type UserDeletedEvent struct {
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	DeletedAt time.Time `json:"deleted_at"`
}

// UserStatusChangedEvent 用户状态变更事件
type UserStatusChangedEvent struct {
	UserID    uint      `json:"user_id"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
	ChangedAt time.Time `json:"changed_at"`
}

// UserRoleChangedEvent 用户角色变更事件
type UserRoleChangedEvent struct {
	UserID    uint      `json:"user_id"`
	OldRole   string    `json:"old_role"`
	NewRole   string    `json:"new_role"`
	ChangedAt time.Time `json:"changed_at"`
}

// UserPasswordChangedEvent 用户密码变更事件
type UserPasswordChangedEvent struct {
	UserID    uint      `json:"user_id"`
	ChangedAt time.Time `json:"changed_at"`
}

// UserEmailVerifiedEvent 用户邮箱验证事件
type UserEmailVerifiedEvent struct {
	UserID     uint      `json:"user_id"`
	Email      string    `json:"email"`
	VerifiedAt time.Time `json:"verified_at"`
}

// UserPhoneVerifiedEvent 用户手机验证事件
type UserPhoneVerifiedEvent struct {
	UserID     uint      `json:"user_id"`
	Phone      string    `json:"phone"`
	VerifiedAt time.Time `json:"verified_at"`
}

// UserLoginEvent 用户登录事件
type UserLoginEvent struct {
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	LoginAt   time.Time `json:"login_at"`
}

// UserLogoutEvent 用户登出事件
type UserLogoutEvent struct {
	UserID   uint      `json:"user_id"`
	Username string    `json:"username"`
	LogoutAt time.Time `json:"logout_at"`
}

// UserFailedLoginEvent 用户登录失败事件
type UserFailedLoginEvent struct {
	Username  string    `json:"username"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	Reason    string    `json:"reason"`
	FailedAt  time.Time `json:"failed_at"`
}

// UserLockedEvent 用户账户锁定事件
type UserLockedEvent struct {
	UserID   uint      `json:"user_id"`
	Username string    `json:"username"`
	Reason   string    `json:"reason"`
	LockedAt time.Time `json:"locked_at"`
}

// UserUnlockedEvent 用户账户解锁事件
type UserUnlockedEvent struct {
	UserID     uint      `json:"user_id"`
	Username   string    `json:"username"`
	UnlockedAt time.Time `json:"unlocked_at"`
}
