package persistence

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/fixgateway/domain"
	"github.com/wyfcoding/pkg/database"
	"gorm.io/gorm"
)

type fixRepository struct {
	*database.GormRepository[domain.FixSession]
}

func NewFixRepository(db *gorm.DB) domain.Repository {
	return &fixRepository{
		GormRepository: database.NewGormRepository[domain.FixSession](db),
	}
}

func (r *fixRepository) Save(s *domain.FixSession) error {
	return r.Upsert(context.Background(), s)
}

func (r *fixRepository) FindByID(id string) (*domain.FixSession, error) {
	var s domain.FixSession
	err := r.DB(context.Background()).Where("session_id = ?", id).First(&s).Error
	return &s, err
}
