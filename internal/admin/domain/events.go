package domain

import "time"

// AdminCreatedEvent 管理员创建事件
type AdminCreatedEvent struct {
	AdminID    uint      `json:"admin_id"`
	Username   string    `json:"username"`
	RoleID     uint      `json:"role_id"`
	Timestamp  time.Time `json:"timestamp"`
}

// AdminLoggedInEvent 管理员登录事件
type AdminLoggedInEvent struct {
	AdminID    uint      `json:"admin_id"`
	Username   string    `json:"username"`
	Timestamp  time.Time `json:"timestamp"`
}

// RoleCreatedEvent 角色创建事件
type RoleCreatedEvent struct {
	RoleID     uint      `json:"role_id"`
	Name       string    `json:"name"`
	Timestamp  time.Time `json:"timestamp"`
}
