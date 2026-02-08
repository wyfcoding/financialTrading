package domain

import (
	"context"
	"time"
)

type PortfolioRepository interface {
	SaveSnapshot(ctx context.Context, s *PortfolioSnapshot) error
	GetSnapshots(ctx context.Context, userID string, start, end time.Time) ([]PortfolioSnapshot, error)

	SavePerformance(ctx context.Context, p *UserPerformance) error
	GetPerformance(ctx context.Context, userID string) (*UserPerformance, error)
}
