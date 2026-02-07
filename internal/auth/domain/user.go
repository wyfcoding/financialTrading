package domain

import "time"

type UserRole string

const (
	RoleAdmin       UserRole = "ADMIN"
	RoleInstitution UserRole = "INSTITUTION"
	RoleTrader      UserRole = "TRADER"
)

type User struct {
	ID           uint      `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         UserRole  `json:"role"`
}

func NewUser(email, passwordHash string) *User {
	return &User{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         RoleTrader,
	}
}
