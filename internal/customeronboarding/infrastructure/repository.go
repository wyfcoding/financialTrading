package infrastructure

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/customeronboarding/domain"
	"gorm.io/gorm"
)

type GormOnboardingRepository struct {
	db *gorm.DB
}

func NewGormOnboardingRepository(db *gorm.DB) *GormOnboardingRepository {
	return &GormOnboardingRepository{db: db}
}

func (r *GormOnboardingRepository) Save(ctx context.Context, app *domain.OnboardingApplication) error {
	return r.db.WithContext(ctx).Save(app).Error
}

func (r *GormOnboardingRepository) Get(ctx context.Context, id string) (*domain.OnboardingApplication, error) {
	var app domain.OnboardingApplication
	err := r.db.WithContext(ctx).Where("application_id = ?", id).First(&app).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &app, err
}

func (r *GormOnboardingRepository) ListByEmail(ctx context.Context, email string) ([]*domain.OnboardingApplication, error) {
	var apps []*domain.OnboardingApplication
	err := r.db.WithContext(ctx).Where("email = ?", email).Find(&apps).Error
	return apps, err
}
