package domain

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// APIKey 为机构客户提供的访问凭证
type APIKey struct {
	gorm.Model
	Key        string `gorm:"column:api_key;type:varchar(64);uniqueIndex;not null" json:"api_key"`
	SecretHash string `gorm:"column:secret_hash;type:varchar(128);not null" json:"-"` // 数据库仅存储哈希
	UserID     string `gorm:"column:user_id;type:varchar(64);index;not null" json:"user_id"`
	Label      string `gorm:"column:label;type:varchar(100)" json:"label"`
	Enabled    bool   `gorm:"column:enabled;default:true" json:"enabled"`
	Scopes     string `gorm:"column:scopes;type:text" json:"scopes"` // JSON 数组或逗号分隔的权限点，如 trade:write,market:read
}

func (k *APIKey) HasScope(scope string) bool {
	// 简单的逗号分隔检查或使用 JSON 解析
	return k.Enabled && (scope == "" || strings.Contains(k.Scopes, scope))
}

type APIKeyRepository interface {
	Save(ctx context.Context, key *APIKey) error
	GetByKey(ctx context.Context, key string) (*APIKey, error)
	ListByUserID(ctx context.Context, userID string) ([]*APIKey, error)
}
