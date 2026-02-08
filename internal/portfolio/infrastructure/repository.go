package infrastructure

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/portfolio/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PortfolioRepository struct {
	db *gorm.DB
}

func NewPortfolioRepository(db *gorm.DB) *PortfolioRepository {
	return &PortfolioRepository{db: db}
}

func (r *PortfolioRepository) SaveSnapshot(ctx context.Context, s *domain.PortfolioSnapshot) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_equity", "currency"}),
	}).Create(s).Error
}

func (r *PortfolioRepository) GetSnapshots(ctx context.Context, userID string, start, end time.Time) ([]domain.PortfolioSnapshot, error) {
	var snaps []domain.PortfolioSnapshot
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND date >= ? AND date <= ?", userID, start, end).
		Order("date ASC").
		Find(&snaps).Error; err != nil {
		return nil, err
	}
	return snaps, nil
}

func (r *PortfolioRepository) SavePerformance(ctx context.Context, p *domain.UserPerformance) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *PortfolioRepository) GetPerformance(ctx context.Context, userID string) (*domain.UserPerformance, error) {
	var p domain.UserPerformance
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}
