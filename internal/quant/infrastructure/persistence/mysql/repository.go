package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"gorm.io/gorm"
)

type signalRepository struct {
	db *gorm.DB
}

func NewSignalRepository(db *gorm.DB) domain.SignalRepository {
	return &signalRepository{db: db}
}

func (r *signalRepository) Save(ctx context.Context, signal *domain.Signal) error {
	return r.db.WithContext(ctx).Create(signal).Error
}

func (r *signalRepository) GetLatest(ctx context.Context, symbol string, indicator domain.IndicatorType, period int) (*domain.Signal, error) {
	var signal domain.Signal
	err := r.db.WithContext(ctx).
		Where("symbol = ? AND indicator = ? AND period = ?", symbol, indicator, period).
		Order("timestamp desc").
		First(&signal).Error
	return &signal, err
}
