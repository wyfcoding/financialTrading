package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/auth/domain"
	"gorm.io/gorm"
)

type apiKeyRepository struct {
	db *gorm.DB
}

func NewAPIKeyRepository(db *gorm.DB) domain.APIKeyRepository {
	return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) Save(ctx context.Context, key *domain.APIKey) error {
	return r.db.WithContext(ctx).Save(key).Error
}

func (r *apiKeyRepository) GetByKey(ctx context.Context, key string) (*domain.APIKey, error) {
	var ak domain.APIKey
	if err := r.db.WithContext(ctx).Where("api_key = ?", key).First(&ak).Error; err != nil {
		return nil, err
	}
	return &ak, nil
}

func (r *apiKeyRepository) ListByUserID(ctx context.Context, userID string) ([]*domain.APIKey, error) {
	var keys []*domain.APIKey
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}
