package domain

import (
	"context"
	"time"
)

// AuthSession 用户认证会话
type AuthSession struct {
	Token     string    `json:"token"`
	UserID    uint      `json:"user_id"`
	Email     string    `json:"email"`
	Role      UserRole  `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *AuthSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// SessionRepository 会话仓储接口（通常仅实现 Redis 版本）
type SessionRepository interface {
	Save(ctx context.Context, session *AuthSession) error
	Get(ctx context.Context, token string) (*AuthSession, error)
	Delete(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID uint) error
}
