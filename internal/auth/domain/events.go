package domain

import "time"

// UserRegisteredEvent 用户注册事件
type UserRegisteredEvent struct {
	UserID    uint      `json:"user_id"`
	Email     string    `json:"email"`
	Role      UserRole  `json:"role"`
	Timestamp time.Time `json:"timestamp"`
}

// UserLoggedInEvent 用户登录事件
type UserLoggedInEvent struct {
	UserID    uint      `json:"user_id"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
}

// APIKeyCreatedEvent API Key 创建事件
type APIKeyCreatedEvent struct {
	APIKeyID  uint      `json:"api_key_id"`
	UserID    string    `json:"user_id"`
	Label     string    `json:"label"`
	Scopes    string    `json:"scopes"`
	Timestamp time.Time `json:"timestamp"`
}

// APIKeyValidatedEvent API Key 验证事件
type APIKeyValidatedEvent struct {
	APIKeyID  uint      `json:"api_key_id"`
	UserID    string    `json:"user_id"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp"`
}
