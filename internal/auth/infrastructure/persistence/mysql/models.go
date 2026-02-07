package mysql

import (
	"time"

	"github.com/wyfcoding/financialtrading/internal/auth/domain"
)

// UserModel MySQL 用户表映射
type UserModel struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
	Email        string    `gorm:"column:email;type:varchar(255);uniqueIndex;not null"`
	PasswordHash string    `gorm:"column:password_hash;type:varchar(255);not null"`
	Role         string    `gorm:"column:role;type:varchar(20);default:'TRADER';not null"`
}

func (UserModel) TableName() string {
	return "users"
}

// APIKeyModel MySQL API Key 表映射
type APIKeyModel struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
	Key        string    `gorm:"column:api_key;type:varchar(64);uniqueIndex;not null"`
	SecretHash string    `gorm:"column:secret_hash;type:varchar(128);not null"`
	UserID     string    `gorm:"column:user_id;type:varchar(64);index;not null"`
	Label      string    `gorm:"column:label;type:varchar(100)"`
	Enabled    bool      `gorm:"column:enabled;default:true"`
	Scopes     string    `gorm:"column:scopes;type:text"`
}

func (APIKeyModel) TableName() string {
	return "api_keys"
}

func toUserModel(user *domain.User) *UserModel {
	if user == nil {
		return nil
	}
	return &UserModel{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         string(user.Role),
	}
}

func toUser(model *UserModel) *domain.User {
	if model == nil {
		return nil
	}
	return &domain.User{
		ID:           model.ID,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
		Email:        model.Email,
		PasswordHash: model.PasswordHash,
		Role:         domain.UserRole(model.Role),
	}
}

func toAPIKeyModel(key *domain.APIKey) *APIKeyModel {
	if key == nil {
		return nil
	}
	return &APIKeyModel{
		ID:         key.ID,
		CreatedAt:  key.CreatedAt,
		UpdatedAt:  key.UpdatedAt,
		Key:        key.Key,
		SecretHash: key.SecretHash,
		UserID:     key.UserID,
		Label:      key.Label,
		Enabled:    key.Enabled,
		Scopes:     key.Scopes,
	}
}

func toAPIKey(model *APIKeyModel) *domain.APIKey {
	if model == nil {
		return nil
	}
	return &domain.APIKey{
		ID:         model.ID,
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
		Key:        model.Key,
		SecretHash: model.SecretHash,
		UserID:     model.UserID,
		Label:      model.Label,
		Enabled:    model.Enabled,
		Scopes:     model.Scopes,
	}
}
