package domain

import (
	"context"
)

type AlternativeDataRepository interface {
	SaveNews(ctx context.Context, news *NewsItem) error
	ListNews(ctx context.Context, symbol string, limit int) ([]*NewsItem, error)

	SaveSentiment(ctx context.Context, s *Sentiment) error
	GetLatestSentiment(ctx context.Context, symbol string) (*Sentiment, error)
}
