package infrastructure

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/alternativedata/domain"
	"gorm.io/gorm"
)

type GormAlternativeDataRepository struct {
	db *gorm.DB
}

func NewGormAlternativeDataRepository(db *gorm.DB) *GormAlternativeDataRepository {
	return &GormAlternativeDataRepository{db: db}
}

func (r *GormAlternativeDataRepository) SaveNews(ctx context.Context, news *domain.NewsItem) error {
	return r.db.WithContext(ctx).Save(news).Error
}

func (r *GormAlternativeDataRepository) ListNews(ctx context.Context, symbol string, limit int) ([]*domain.NewsItem, error) {
	var news []*domain.NewsItem
	query := r.db.WithContext(ctx)
	if symbol != "" {
		query = query.Where("symbol = ?", symbol)
	}
	err := query.Order("published_at desc").Limit(limit).Find(&news).Error
	return news, err
}

func (r *GormAlternativeDataRepository) SaveSentiment(ctx context.Context, s *domain.Sentiment) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *GormAlternativeDataRepository) GetLatestSentiment(ctx context.Context, symbol string) (*domain.Sentiment, error) {
	var s domain.Sentiment
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("timestamp desc").First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &s, err
}
