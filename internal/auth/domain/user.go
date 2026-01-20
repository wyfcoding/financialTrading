package domain

import "gorm.io/gorm"

type UserRole string

const (
	RoleAdmin       UserRole = "ADMIN"
	RoleInstitution UserRole = "INSTITUTION"
	RoleTrader      UserRole = "TRADER"
)

type User struct {
	gorm.Model
	Email        string   `gorm:"column:email;type:varchar(255);uniqueIndex;not null"`
	PasswordHash string   `gorm:"column:password_hash;type:varchar(255);not null"`
	Role         UserRole `gorm:"column:role;type:varchar(20);default:'TRADER';not null"`
}

func (User) TableName() string { return "users" }

func NewUser(email, passwordHash string) *User {
	return &User{Email: email, PasswordHash: passwordHash}
}
