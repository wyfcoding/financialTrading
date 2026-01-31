package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/user/domain"
	"gorm.io/gorm"
)

type userRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Save(ctx context.Context, user *domain.UserProfile) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) GetByID(ctx context.Context, id uint) (*domain.UserProfile, error) {
	var u domain.UserProfile
	err := r.db.WithContext(ctx).First(&u, id).Error
	return &u, err
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.UserProfile, error) {
	var u domain.UserProfile
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	return &u, err
}
