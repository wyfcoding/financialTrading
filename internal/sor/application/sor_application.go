package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/sor/domain"
)

type SORApplicationService struct {
	engine        domain.SOREngine
	planRepo      SORPlanRepository
	executionRepo ExecutionRepository
	venueRepo     VenueRepository
	logger        *slog.Logger
}

type SORPlanRepository interface {
	Save(ctx context.Context, plan *domain.SORPlan) error
	Get(ctx context.Context, planID string) (*domain.SORPlan, error)
	GetByParentOrder(ctx context.Context, parentOrderID string) (*domain.SORPlan, error)
	Update(ctx context.Context, plan *domain.SORPlan) error
}

type ExecutionRepository interface {
	Save(ctx context.Context, report *domain.ExecutionReport) error
	GetByPlanID(ctx context.Context, planID string) ([]*domain.ExecutionReport, error)
	GetByRouteID(ctx context.Context, routeID string) (*domain.ExecutionReport, error)
}

type VenueRepository interface {
	GetAll(ctx context.Context) ([]*domain.Venue, error)
	GetByID(ctx context.Context, venueID string) (*domain.Venue, error)
	GetActive(ctx context.Context) ([]*domain.Venue, error)
}

func NewSORApplicationService(
	engine domain.SOREngine,
	planRepo SORPlanRepository,
	executionRepo ExecutionRepository,
	venueRepo VenueRepository,
	logger *slog.Logger,
) *SORApplicationService {
	return &SORApplicationService{
		engine:        engine,
		planRepo:      planRepo,
		executionRepo: executionRepo,
		venueRepo:     venueRepo,
		logger:        logger,
	}
}

type CreateSORPlanCommand struct {
	ParentOrderID string
	Symbol        string
	Side          domain.OrderSide
	Quantity      int64
	Strategy      domain.RoutingStrategy
	MaxVenues     int
	MinFillRate   float64
	MaxSlippage   float64
	TimeLimit     time.Duration
	AllowDarkPool bool
	VenueFilter   []string
}

func (s *SORApplicationService) CreateSORPlan(ctx context.Context, cmd CreateSORPlanCommand) (*domain.SORPlan, error) {
	s.logger.InfoContext(ctx, "creating SOR plan",
		"symbol", cmd.Symbol,
		"side", cmd.Side,
		"quantity", cmd.Quantity,
		"strategy", cmd.Strategy,
	)
	
	if cmd.TimeLimit == 0 {
		cmd.TimeLimit = 5 * time.Minute
	}
	if cmd.MaxVenues == 0 {
		cmd.MaxVenues = 5
	}
	if cmd.MinFillRate == 0 {
		cmd.MinFillRate = 0.95
	}
	
	req := &domain.RoutingRequest{
		RequestID:     fmt.Sprintf("REQ-%d", time.Now().UnixNano()),
		Symbol:        cmd.Symbol,
		Side:          cmd.Side,
		Quantity:      cmd.Quantity,
		Strategy:      cmd.Strategy,
		MaxVenues:     cmd.MaxVenues,
		MinFillRate:   cmd.MinFillRate,
		MaxSlippage:   cmd.MaxSlippage,
		TimeLimit:     cmd.TimeLimit,
		AllowDarkPool: cmd.AllowDarkPool,
		VenueFilter:   cmd.VenueFilter,
	}
	
	plan, err := s.engine.CreateRoutingPlan(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create routing plan: %w", err)
	}
	
	plan.ParentOrderID = cmd.ParentOrderID
	
	optimizedPlan, err := s.engine.OptimizeRoutes(ctx, plan)
	if err != nil {
		s.logger.WarnContext(ctx, "route optimization failed, using original plan", "error", err)
	} else {
		plan = optimizedPlan
	}
	
	impact, err := s.engine.EstimateMarketImpact(ctx, cmd.Symbol, cmd.Side, cmd.Quantity)
	if err == nil {
		plan.MarketImpact = impact
	}
	
	plan.ConfidenceScore = s.calculateConfidenceScore(plan)
	
	if s.planRepo != nil {
		if err := s.planRepo.Save(ctx, plan); err != nil {
			s.logger.ErrorContext(ctx, "failed to save plan", "error", err)
		}
	}
	
	s.logger.InfoContext(ctx, "SOR plan created",
		"plan_id", plan.PlanID,
		"routes", len(plan.Routes),
		"avg_price", plan.AveragePrice,
		"total_fee", plan.TotalFee,
	)
	
	return plan, nil
}

type ExecuteRouteCommand struct {
	PlanID  string
	RouteID string
}

func (s *SORApplicationService) ExecuteRoute(ctx context.Context, cmd ExecuteRouteCommand) (*domain.ExecutionReport, error) {
	plan, err := s.planRepo.Get(ctx, cmd.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	
	var route *domain.OrderRoute
	for _, r := range plan.Routes {
		if r.RouteID == cmd.RouteID {
			route = &r
			break
		}
	}
	
	if route == nil {
		return nil, fmt.Errorf("route not found: %s", cmd.RouteID)
	}
	
	startTime := time.Now()
	
	report := &domain.ExecutionReport{
		ReportID:    fmt.Sprintf("EXEC-%d", time.Now().UnixNano()),
		PlanID:      cmd.PlanID,
		RouteID:     cmd.RouteID,
		VenueID:     route.VenueID,
		FilledQty:   route.Quantity,
		FilledPrice: route.Price,
		Fee:         route.Fee,
		ExecutedAt:  time.Now(),
	}
	
	report.Latency = time.Since(startTime)
	report.Slippage = s.calculateSlippage(route.Price, plan.AveragePrice)
	
	if s.executionRepo != nil {
		if err := s.executionRepo.Save(ctx, report); err != nil {
			s.logger.ErrorContext(ctx, "failed to save execution report", "error", err)
		}
	}
	
	s.logger.InfoContext(ctx, "route executed",
		"plan_id", cmd.PlanID,
		"route_id", cmd.RouteID,
		"filled_qty", report.FilledQty,
		"filled_price", report.FilledPrice,
		"latency", report.Latency,
	)
	
	return report, nil
}

type UpdatePlanStatusCommand struct {
	PlanID string
	Status domain.PlanStatus
}

func (s *SORApplicationService) UpdatePlanStatus(ctx context.Context, cmd UpdatePlanStatusCommand) error {
	plan, err := s.planRepo.Get(ctx, cmd.PlanID)
	if err != nil {
		return fmt.Errorf("failed to get plan: %w", err)
	}
	
	plan.Status = cmd.Status
	
	if s.planRepo != nil {
		if err := s.planRepo.Update(ctx, plan); err != nil {
			return fmt.Errorf("failed to update plan: %w", err)
		}
	}
	
	s.logger.InfoContext(ctx, "plan status updated",
		"plan_id", cmd.PlanID,
		"status", cmd.Status,
	)
	
	return nil
}

type GetPlanQuery struct {
	PlanID string
}

func (s *SORApplicationService) GetPlan(ctx context.Context, query GetPlanQuery) (*domain.SORPlan, error) {
	if s.planRepo == nil {
		return nil, fmt.Errorf("plan repository not configured")
	}
	return s.planRepo.Get(ctx, query.PlanID)
}

type GetExecutionReportsQuery struct {
	PlanID string
}

func (s *SORApplicationService) GetExecutionReports(ctx context.Context, query GetExecutionReportsQuery) ([]*domain.ExecutionReport, error) {
	if s.executionRepo == nil {
		return nil, fmt.Errorf("execution repository not configured")
	}
	return s.executionRepo.GetByPlanID(ctx, query.PlanID)
}

type GetLiquidityScoreQuery struct {
	VenueID string
	Symbol  string
}

func (s *SORApplicationService) GetLiquidityScore(ctx context.Context, query GetLiquidityScoreQuery) (*domain.LiquidityScore, error) {
	return s.engine.CalculateLiquidityScore(ctx, query.VenueID, query.Symbol)
}

type EstimateImpactQuery struct {
	Symbol   string
	Side     domain.OrderSide
	Quantity int64
}

func (s *SORApplicationService) EstimateMarketImpact(ctx context.Context, query EstimateImpactQuery) (float64, error) {
	return s.engine.EstimateMarketImpact(ctx, query.Symbol, query.Side, query.Quantity)
}

type GetVenuesQuery struct {
	ActiveOnly bool
}

func (s *SORApplicationService) GetVenues(ctx context.Context, query GetVenuesQuery) ([]*domain.Venue, error) {
	if s.venueRepo == nil {
		return nil, fmt.Errorf("venue repository not configured")
	}
	
	if query.ActiveOnly {
		return s.venueRepo.GetActive(ctx)
	}
	return s.venueRepo.GetAll(ctx)
}

type GetDepthsQuery struct {
	Symbol string
	Venues []string
}

func (s *SORApplicationService) GetDepths(ctx context.Context, query GetDepthsQuery) ([]*domain.MarketDepth, error) {
	return s.engine.AggregateDepths(ctx, query.Symbol, query.Venues)
}

func (s *SORApplicationService) calculateSlippage(actualPrice, expectedPrice float64) float64 {
	if expectedPrice == 0 {
		return 0
	}
	return (actualPrice - expectedPrice) / expectedPrice
}

func (s *SORApplicationService) calculateConfidenceScore(plan *domain.SORPlan) float64 {
	if len(plan.Routes) == 0 {
		return 0
	}
	
	totalExpectedFill := 0.0
	for _, r := range plan.Routes {
		totalExpectedFill += r.ExpectedFill
	}
	
	avgExpectedFill := totalExpectedFill / float64(len(plan.Routes))
	
	venueDiversity := float64(len(plan.Routes)) / 5.0
	if venueDiversity > 1.0 {
		venueDiversity = 1.0
	}
	
	feeScore := 1.0 - (plan.TotalFee / plan.ExpectedCost)
	if feeScore < 0 {
		feeScore = 0
	}
	
	score := avgExpectedFill*0.4 + venueDiversity*0.3 + feeScore*0.3
	
	return score
}
