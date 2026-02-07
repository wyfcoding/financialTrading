package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/auth/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type apiKeyRepository struct {
	db *gorm.DB
}

func NewAPIKeyRepository(db *gorm.DB) domain.APIKeyRepository {
	return &apiKeyRepository{db: db}
}

// --- tx helpers ---

func (r *apiKeyRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *apiKeyRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *apiKeyRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *apiKeyRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *apiKeyRepository) Save(ctx context.Context, key *domain.APIKey) error {
	db := r.getDB(ctx)
	model := toAPIKeyModel(key)
	if model.ID == 0 {
		if err := db.WithContext(ctx).Create(model).Error; err != nil {
			return err
		}
		key.ID = model.ID
		key.CreatedAt = model.CreatedAt
		key.UpdatedAt = model.UpdatedAt
		return nil
	}

	return db.WithContext(ctx).
		Model(&APIKeyModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"api_key":     model.Key,
			"secret_hash": model.SecretHash,
			"user_id":     model.UserID,
			"label":       model.Label,
			"enabled":     model.Enabled,
			"scopes":      model.Scopes,
		}).Error
}

func (r *apiKeyRepository) GetByKey(ctx context.Context, key string) (*domain.APIKey, error) {
	var model APIKeyModel
	if err := r.getDB(ctx).WithContext(ctx).Where("api_key = ?", key).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toAPIKey(&model), nil
}

func (r *apiKeyRepository) ListByUserID(ctx context.Context, userID string) ([]*domain.APIKey, error) {
	var models []*APIKeyModel
	if err := r.getDB(ctx).WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, err
	}
	keys := make([]*domain.APIKey, len(models))
	for i, model := range models {
		keys[i] = toAPIKey(model)
	}
	return keys, nil
}

func (r *apiKeyRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
