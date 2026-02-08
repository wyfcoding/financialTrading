package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"gorm.io/gorm"
)

type venueRepository struct {
	db *gorm.DB
}

func NewVenueRepository(db *gorm.DB) domain.VenueRepository {
	return &venueRepository{db: db}
}

func (r *venueRepository) List(ctx context.Context) ([]*domain.Venue, error) {
	var models []VenueModel
	if err := r.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}

	venues := make([]*domain.Venue, 0, len(models))
	for _, m := range models {
		venues = append(venues, toVenue(&m))
	}
	return venues, nil
}

func (r *venueRepository) Get(ctx context.Context, venueID string) (*domain.Venue, error) {
	var m VenueModel
	if err := r.db.WithContext(ctx).Where("venue_id = ?", venueID).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return toVenue(&m), nil
}
