package mysql

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/portfolio/domain"
	"gorm.io/gorm"
)

type PortfolioRepo struct {
	db *gorm.DB
}

func NewPortfolioRepo(db *gorm.DB) *PortfolioRepo {
	return &PortfolioRepo{db: db}
}

func (r *PortfolioRepo) SaveSnapshot(ctx context.Context, s *domain.PortfolioSnapshot) error {
	// Upsert based on UserID + Date
	// But GORM upsert syntax depends on implementation.
	// We can just check exist.
	var exist domain.PortfolioSnapshot
	if err := r.db.WithContext(ctx).Where("user_id = ? AND snapshot_date = ?", s.UserID, s.SnapshotDate.Format("2006-01-02")).First(&exist).Error; err == nil {
		s.ID = exist.ID
		s.CreatedAt = exist.CreatedAt
	}
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *PortfolioRepo) GetLatestSnapshot(ctx context.Context, userID string) (*domain.PortfolioSnapshot, error) {
	var s domain.PortfolioSnapshot
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("snapshot_date desc").First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PortfolioRepo) GetSnapshots(ctx context.Context, userID string, start, end time.Time) ([]domain.PortfolioSnapshot, error) {
	var snapshots []domain.PortfolioSnapshot
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND snapshot_date >= ? AND snapshot_date <= ?", userID, start, end).
		Order("snapshot_date asc").
		Find(&snapshots).Error
	return snapshots, err
}

func (r *PortfolioRepo) ListSnapshots(ctx context.Context, userID string, limit int) ([]*domain.PortfolioSnapshot, error) {
	var snapshots []*domain.PortfolioSnapshot
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("snapshot_date desc").Limit(limit).Find(&snapshots).Error
	return snapshots, err
}

func (r *PortfolioRepo) SavePerformance(ctx context.Context, p *domain.UserPerformance) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *PortfolioRepo) GetPerformance(ctx context.Context, userID string) (*domain.UserPerformance, error) {
	var p domain.UserPerformance
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}
