package domain

import (
	"context"
	"strings"
	"time"
)

// APIKey 为机构客户提供的访问凭证
type APIKey struct {
	ID         uint      `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Key        string    `json:"api_key"`
	SecretHash string    `json:"-"` // 数据库仅存储哈希
	UserID     string    `json:"user_id"`
	Label      string    `json:"label"`
	Enabled    bool      `json:"enabled"`
	Scopes     string    `json:"scopes"` // JSON 数组或逗号分隔的权限点，如 trade:write,market:read
}

func (k *APIKey) HasScope(scope string) bool {
	// 简单的逗号分隔检查或使用 JSON 解析
	return k.Enabled && (scope == "" || strings.Contains(k.Scopes, scope))
}

type APIKeyRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	Save(ctx context.Context, key *APIKey) error
	GetByKey(ctx context.Context, key string) (*APIKey, error)
	ListByUserID(ctx context.Context, userID string) ([]*APIKey, error)
}

// APIKeyRedisRepository 为 API Key 提供高性能缓存
type APIKeyRedisRepository interface {
	Save(ctx context.Context, key *APIKey) error
	Get(ctx context.Context, key string) (*APIKey, error)
	Delete(ctx context.Context, key string) error
}
