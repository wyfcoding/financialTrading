package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/portfolio/domain"
)

type PortfolioQueryService struct {
	portfolioRepo domain.PortfolioRepository
	positionRepo  domain.PositionRepository
	eventRepo     domain.PortfolioEventRepository
	priceService  PriceService
	logger        *slog.Logger
}

type PriceService interface {
	GetCurrentPrice(ctx context.Context, symbol string) (decimal.Decimal, error)
	GetHistoricalPrices(ctx context.Context, symbol string, start, end time.Time) ([]decimal.Decimal, error)
}

func NewPortfolioQueryService(
	portfolioRepo domain.PortfolioRepository,
	positionRepo domain.PositionRepository,
	eventRepo domain.PortfolioEventRepository,
	priceService PriceService,
	logger *slog.Logger,
) *PortfolioQueryService {
	return &PortfolioQueryService{
		portfolioRepo: portfolioRepo,
		positionRepo:  positionRepo,
		eventRepo:     eventRepo,
		priceService:  priceService,
		logger:        logger,
	}
}

func (s *PortfolioQueryService) GetPortfolioOverview(ctx context.Context, userID, currency string) (*domain.PortfolioOverview, error) {
	positions, err := s.positionRepo.ListNonEmpty(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get positions", "error", err, "user_id", userID)
		return nil, err
	}

	overview := domain.NewPortfolioOverview(userID, currency)

	for _, pos := range positions {
		currentPrice, err := s.priceService.GetCurrentPrice(ctx, pos.Symbol)
		if err != nil {
			s.logger.Warn("failed to get current price", "symbol", pos.Symbol, "error", err)
			currentPrice = pos.AvgCost
		}

		pos.UpdateUnrealizedPnL(currentPrice)
		overview.AddPosition(pos)
	}

	overview.CalculateTotalEquity()

	previousSnapshot, err := s.portfolioRepo.GetLatestSnapshot(ctx, userID)
	if err == nil && previousSnapshot != nil {
		overview.UpdateDailyPnL(previousSnapshot.TotalValue)
	}

	overview.LastUpdated = time.Now()

	return overview, nil
}

func (s *PortfolioQueryService) GetPositions(ctx context.Context, userID string) ([]*domain.Position, error) {
	positions, err := s.positionRepo.ListNonEmpty(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, pos := range positions {
		currentPrice, err := s.priceService.GetCurrentPrice(ctx, pos.Symbol)
		if err != nil {
			continue
		}
		pos.UpdateUnrealizedPnL(currentPrice)
	}

	return positions, nil
}

func (s *PortfolioQueryService) GetPosition(ctx context.Context, userID, symbol string) (*domain.Position, error) {
	position, err := s.positionRepo.GetByUserAndSymbol(ctx, userID, symbol)
	if err != nil {
		return nil, err
	}

	currentPrice, err := s.priceService.GetCurrentPrice(ctx, symbol)
	if err == nil {
		position.UpdateUnrealizedPnL(currentPrice)
	}

	return position, nil
}

func (s *PortfolioQueryService) GetPerformanceHistory(ctx context.Context, userID string, days int) ([]*domain.PortfolioSnapshot, error) {
	return s.portfolioRepo.ListSnapshots(ctx, userID, days)
}

func (s *PortfolioQueryService) GetPerformanceMetrics(ctx context.Context, userID string) (*domain.UserPerformance, error) {
	return s.portfolioRepo.GetPerformance(ctx, userID)
}

func (s *PortfolioQueryService) GetPortfolioEvents(ctx context.Context, userID string, limit int) ([]*domain.PortfolioEvent, error) {
	return s.eventRepo.ListByUser(ctx, userID, limit)
}

type PortfolioCommandService struct {
	portfolioRepo domain.PortfolioRepository
	positionRepo  domain.PositionRepository
	eventRepo     domain.PortfolioEventRepository
	logger        *slog.Logger
}

func NewPortfolioCommandService(
	portfolioRepo domain.PortfolioRepository,
	positionRepo domain.PositionRepository,
	eventRepo domain.PortfolioEventRepository,
	logger *slog.Logger,
) *PortfolioCommandService {
	return &PortfolioCommandService{
		portfolioRepo: portfolioRepo,
		positionRepo:  positionRepo,
		eventRepo:     eventRepo,
		logger:        logger,
	}
}

func (s *PortfolioCommandService) OpenPosition(ctx context.Context, userID, symbol string, qty, price decimal.Decimal, posType string) (*domain.Position, error) {
	existingPos, err := s.positionRepo.GetByUserAndSymbol(ctx, userID, symbol)
	if err != nil && err != ErrPositionNotFound {
		return nil, err
	}

	var position *domain.Position
	if existingPos != nil {
		existingPos.AddQuantity(qty, price)
		position = existingPos
	} else {
		position = domain.NewPosition(userID, symbol, qty, price, posType)
	}

	if err := s.positionRepo.Save(ctx, position); err != nil {
		return nil, err
	}

	event := &domain.PortfolioEvent{
		UserID:    userID,
		EventType: domain.EventTypePositionOpen,
		Symbol:    symbol,
		Quantity:  qty,
		Price:     price,
		Timestamp: time.Now(),
	}
	_ = s.eventRepo.Save(ctx, event)

	return position, nil
}

func (s *PortfolioCommandService) ClosePosition(ctx context.Context, userID, symbol string, qty, price decimal.Decimal) (decimal.Decimal, error) {
	position, err := s.positionRepo.GetByUserAndSymbol(ctx, userID, symbol)
	if err != nil {
		return decimal.Zero, err
	}

	if position.AvailableQty.LessThan(qty) {
		return decimal.Zero, ErrInsufficientQuantity
	}

	realizedPnL := position.ReduceQuantity(qty, price)

	if position.IsEmpty() {
		if err := s.positionRepo.Delete(ctx, position.ID); err != nil {
			s.logger.Warn("failed to delete empty position", "error", err)
		}
	} else {
		if err := s.positionRepo.Save(ctx, position); err != nil {
			return decimal.Zero, err
		}
	}

	event := &domain.PortfolioEvent{
		UserID:    userID,
		EventType: domain.EventTypePositionClose,
		Symbol:    symbol,
		Quantity:  qty,
		Price:     price,
		Timestamp: time.Now(),
	}
	_ = s.eventRepo.Save(ctx, event)

	return realizedPnL, nil
}

func (s *PortfolioCommandService) FreezePosition(ctx context.Context, userID, symbol string, qty decimal.Decimal) error {
	position, err := s.positionRepo.GetByUserAndSymbol(ctx, userID, symbol)
	if err != nil {
		return err
	}

	if !position.Freeze(qty) {
		return ErrInsufficientQuantity
	}

	return s.positionRepo.Save(ctx, position)
}

func (s *PortfolioCommandService) UnfreezePosition(ctx context.Context, userID, symbol string, qty decimal.Decimal) error {
	position, err := s.positionRepo.GetByUserAndSymbol(ctx, userID, symbol)
	if err != nil {
		return err
	}

	if !position.Unfreeze(qty) {
		return ErrInvalidQuantity
	}

	return s.positionRepo.Save(ctx, position)
}

func (s *PortfolioCommandService) SaveDailySnapshot(ctx context.Context, userID string, equity decimal.Decimal, currency string) error {
	today := time.Now().Truncate(24 * time.Hour)
	snapshot := domain.NewPortfolioSnapshot(userID, today, equity, currency)
	return s.portfolioRepo.SaveSnapshot(ctx, snapshot)
}

func (s *PortfolioCommandService) UpdatePerformanceMetrics(ctx context.Context, userID string, totalReturn, sharpe, maxDrawdown decimal.Decimal) error {
	perf := &domain.UserPerformance{
		UserID:      userID,
		TotalReturn: totalReturn,
		SharpeRatio: sharpe,
		MaxDrawdown: maxDrawdown,
		UpdatedAt:   time.Now(),
	}
	return s.portfolioRepo.SavePerformance(ctx, perf)
}

var (
	ErrPositionNotFound     = NewPortfolioError("position not found")
	ErrInsufficientQuantity = NewPortfolioError("insufficient quantity")
	ErrInvalidQuantity      = NewPortfolioError("invalid quantity")
)

type PortfolioError struct {
	message string
}

func NewPortfolioError(msg string) *PortfolioError {
	return &PortfolioError{message: msg}
}

func (e *PortfolioError) Error() string {
	return e.message
}

type PortfolioPositionView struct {
	Symbol       string
	Quantity     float64
	AvgPrice     float64
	CurrentPrice float64
	Type         string
}

type PortfolioSnapshotView struct {
	Timestamp string
	Equity    float64
}

type PortfolioAppService struct {
	portfolioRepo domain.PortfolioRepository
	logger        *slog.Logger
}

func NewPortfolioAppService(repo domain.PortfolioRepository, logger *slog.Logger) *PortfolioAppService {
	return &PortfolioAppService{
		portfolioRepo: repo,
		logger:        logger,
	}
}

func (s *PortfolioAppService) GetPortfolio(ctx context.Context, userID, currency string) (float64, float64, float64, float64, error) {
	snapshot, err := s.portfolioRepo.GetLatestSnapshot(ctx, userID)
	if err != nil || snapshot == nil {
		return 0, 0, 0, 0, err
	}

	totalEquity, _ := snapshot.TotalValue.Float64()
	dailyPnLPct, _ := snapshot.DayReturn.Float64()
	return totalEquity, 0, 0, dailyPnLPct, nil
}

func (s *PortfolioAppService) GetPositions(ctx context.Context, userID string) ([]PortfolioPositionView, error) {
	// Stage A: this entrypoint keeps cmd/portfolio buildable even when position read model is not wired.
	return []PortfolioPositionView{}, nil
}

func (s *PortfolioAppService) GetPerformance(ctx context.Context, userID, timeframe string) ([]PortfolioSnapshotView, float64, float64, float64, error) {
	limit := timeframeToLimit(timeframe)
	snapshots, err := s.portfolioRepo.ListSnapshots(ctx, userID, limit)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	history := make([]PortfolioSnapshotView, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot == nil {
			continue
		}
		equity, _ := snapshot.TotalValue.Float64()
		history = append(history, PortfolioSnapshotView{
			Timestamp: snapshot.SnapshotDate.Format("2006-01-02"),
			Equity:    equity,
		})
	}

	var totalReturn, sharpeRatio, maxDrawdown float64
	if perf, perfErr := s.portfolioRepo.GetPerformance(ctx, userID); perfErr == nil && perf != nil {
		totalReturn, _ = perf.TotalReturn.Float64()
		sharpeRatio, _ = perf.SharpeRatio.Float64()
		maxDrawdown, _ = perf.MaxDrawdown.Float64()
	}

	return history, totalReturn, sharpeRatio, maxDrawdown, nil
}

func timeframeToLimit(timeframe string) int {
	switch timeframe {
	case "1D":
		return 1
	case "1W":
		return 7
	case "1M":
		return 30
	case "3M":
		return 90
	case "6M":
		return 180
	case "1Y":
		return 365
	default:
		return 30
	}
}
