package domain

import (
	"context"
	"time"
)

type PortfolioRepository interface {
	SaveSnapshot(ctx context.Context, s *PortfolioSnapshot) error
	GetSnapshots(ctx context.Context, userID string, start, end time.Time) ([]PortfolioSnapshot, error)
	GetLatestSnapshot(ctx context.Context, userID string) (*PortfolioSnapshot, error)
	ListSnapshots(ctx context.Context, userID string, limit int) ([]*PortfolioSnapshot, error)

	SavePerformance(ctx context.Context, p *UserPerformance) error
	GetPerformance(ctx context.Context, userID string) (*UserPerformance, error)
}

type PositionRepository interface {
	Save(ctx context.Context, position *Position) error
	GetByID(ctx context.Context, id uint) (*Position, error)
	GetByUserAndSymbol(ctx context.Context, userID, symbol string) (*Position, error)
	ListByUser(ctx context.Context, userID string) ([]*Position, error)
	ListNonEmpty(ctx context.Context, userID string) ([]*Position, error)
	Delete(ctx context.Context, id uint) error
	BatchSave(ctx context.Context, positions []*Position) error
}

type PortfolioEventRepository interface {
	Save(ctx context.Context, event *PortfolioEvent) error
	ListByUser(ctx context.Context, userID string, limit int) ([]*PortfolioEvent, error)
}
