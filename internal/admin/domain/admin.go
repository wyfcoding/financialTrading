package domain

import (
	"gorm.io/gorm"
)

type Admin struct {
	gorm.Model
	Username     string `gorm:"column:username;type:varchar(50);uniqueIndex;not null"`
	PasswordHash string `gorm:"column:password_hash;type:varchar(255);not null"`
	RoleID       uint   `gorm:"column:role_id;not null"`
	Role         Role   `gorm:"foreignKey:RoleID"`
}

func (Admin) TableName() string {
	return "admins"
}

type Role struct {
	gorm.Model
	Name        string `gorm:"column:name;type:varchar(50);uniqueIndex;not null"`
	Permissions string `gorm:"column:permissions;type:json"` // Simple JSON storage for permissions list
}

func (Role) TableName() string {
	return "roles"
}

func NewAdmin(username, passwordHash string, roleID uint) *Admin {
	return &Admin{
		Username:     username,
		PasswordHash: passwordHash,
		RoleID:       roleID,
	}
}

func NewRole(name string, permissions string) *Role {
	return &Role{
		Name:        name,
		Permissions: permissions,
	}
}
