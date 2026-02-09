package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/auth/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type userRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

// --- tx helpers ---

func (r *userRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *userRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *userRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *userRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *userRepository) Save(ctx context.Context, user *domain.User) error {
	db := r.getDB(ctx)
	model := toUserModel(user)
	if model.ID == 0 {
		if err := db.WithContext(ctx).Create(model).Error; err != nil {
			return err
		}
		user.ID = model.ID
		user.CreatedAt = model.CreatedAt
		user.UpdatedAt = model.UpdatedAt
		return nil
	}

	return db.WithContext(ctx).
		Model(&UserModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"email":         model.Email,
			"password_hash": model.PasswordHash,
			"role":          model.Role,
			"source":        model.Source,
		}).Error
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var model UserModel
	err := r.getDB(ctx).WithContext(ctx).Where("email = ?", email).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toUser(&model), nil
}

func (r *userRepository) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	var model UserModel
	err := r.getDB(ctx).WithContext(ctx).First(&model, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toUser(&model), nil
}

func (r *userRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
