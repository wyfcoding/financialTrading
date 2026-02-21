package application

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/portfolio/domain"
	"gorm.io/gorm"
)

type PositionView struct {
	Symbol       string
	Quantity     float64
	AvgPrice     float64
	CurrentPrice float64
	Type         string
}

type PerformancePoint struct {
	Timestamp string
	Equity    float64
}

type PortfolioAppService struct {
	repo   domain.PortfolioRepository
	logger *slog.Logger
}

func NewPortfolioAppService(repo domain.PortfolioRepository, logger *slog.Logger) *PortfolioAppService {
	return &PortfolioAppService{repo: repo, logger: logger}
}

func (s *PortfolioAppService) GetPortfolio(ctx context.Context, userID, currency string) (float64, float64, float64, float64, error) {
	snap, err := s.repo.GetLatestSnapshot(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, 0, 0, nil
		}
		return 0, 0, 0, 0, err
	}
	return snap.TotalEquity.InexactFloat64(),
		snap.UnrealizedPnL.InexactFloat64(),
		snap.RealizedPnL.InexactFloat64(),
		snap.DailyPnLPct.InexactFloat64(),
		nil
}

func (s *PortfolioAppService) GetPositions(ctx context.Context, userID string) ([]*PositionView, error) {
	type positionLister interface {
		ListByUser(ctx context.Context, userID string) ([]*domain.Position, error)
	}
	lister, ok := s.repo.(positionLister)
	if !ok {
		return []*PositionView{}, nil
	}
	positions, err := lister.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*PositionView, 0, len(positions))
	for _, p := range positions {
		out = append(out, &PositionView{
			Symbol:       p.Symbol,
			Quantity:     p.Quantity.InexactFloat64(),
			AvgPrice:     p.AvgCost.InexactFloat64(),
			CurrentPrice: p.AvgCost.InexactFloat64(),
			Type:         p.PositionType,
		})
	}
	return out, nil
}

func (s *PortfolioAppService) GetPerformance(ctx context.Context, userID, timeframe string) ([]*PerformancePoint, float64, float64, float64, error) {
	start, end := timeframeRange(timeframe)
	snaps, err := s.repo.GetSnapshots(ctx, userID, start, end)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 0, 0, 0, err
	}

	points := make([]*PerformancePoint, 0, len(snaps))
	for _, snap := range snaps {
		points = append(points, &PerformancePoint{
			Timestamp: snap.SnapshotDate.Format("2006-01-02"),
			Equity:    snap.TotalEquity.InexactFloat64(),
		})
	}

	perf, err := s.repo.GetPerformance(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return points, 0, 0, 0, nil
		}
		return nil, 0, 0, 0, err
	}
	return points, perf.TotalReturn.InexactFloat64(), perf.SharpeRatio.InexactFloat64(), perf.MaxDrawdown.InexactFloat64(), nil
}

func timeframeRange(timeframe string) (time.Time, time.Time) {
	now := time.Now()
	switch timeframe {
	case "1D":
		return now.Add(-24 * time.Hour), now
	case "1W":
		return now.AddDate(0, 0, -7), now
	case "1M":
		return now.AddDate(0, -1, 0), now
	case "3M":
		return now.AddDate(0, -3, 0), now
	case "1Y":
		return now.AddDate(-1, 0, 0), now
	case "YTD":
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()), now
	default:
		return time.Unix(0, 0), now
	}
}
