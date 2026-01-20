package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
	"gorm.io/gorm"
)

type referenceRepository struct {
	db *gorm.DB
}

func NewReferenceRepository(db *gorm.DB) domain.ReferenceRepository {
	return &referenceRepository{db: db}
}

func (r *referenceRepository) Save(ctx context.Context, instrument *domain.Instrument) error {
	return r.db.WithContext(ctx).Save(instrument).Error
}

func (r *referenceRepository) GetInstrument(ctx context.Context, symbol string) (*domain.Instrument, error) {
	var instr domain.Instrument
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&instr).Error
	return &instr, err
}

func (r *referenceRepository) ListInstruments(ctx context.Context) ([]*domain.Instrument, error) {
	var instruments []*domain.Instrument
	err := r.db.WithContext(ctx).Find(&instruments).Error
	return instruments, err
}
